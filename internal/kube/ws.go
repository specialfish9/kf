// THIS FILE CONTAINS EXPERIMENTAL CODE AND MAY CHANGE AT ANY TIME.
package kube

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	v2 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"
)

func (kl *kubeLayer) forwardWS(req PortForwardRequest) error {
	slog.Debug("Requesting port-forward via WebSocket", "localPort", req.LocalPort, "remotePort", req.RemotePort, "service", req.Service.Name)
	config := kl.config

	pods, err := kl.getPods(req.Service)
	if err != nil {
		return err
	}

	if len(pods) == 0 {
		return fmt.Errorf("no pods found for service %s", req.Service.Name)
	}
	slog.Debug("Found pods for service", "podCount", len(pods))

	var pod *v2.Pod
	for _, p := range pods {
		if p.Status.Phase == "Running" {
			pod = &p
			break
		}
	}
	if pod == nil {
		return fmt.Errorf("no running pods found for service %s", req.Service.Name)
	}
	slog.Debug("Selected pod for port-forward", "podName", pod.Name, "namespace", pod.Namespace)

	namespace := pod.Namespace
	podName := pod.Name
	port := strconv.Itoa(req.RemotePort)

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, podName)

	// Build the WebSocket URL
	u, err := constructUrl("wss", config.Host[8:], path)
	if err != nil {
		return fmt.Errorf("forward: failed to create url: %v", err)
	}

	// Add query params for the ports
	q := u.Query()
	q.Set("ports", port)
	u.RawQuery = q.Encode()

	// Prepare TLS config from kubeconfig
	tlsConfig, err := rest.TLSConfigFor(config)
	if err != nil {
		return fmt.Errorf("forward: failed to create transport: %v", err)
	}

	// Dial WebSocket
	dialer := websocket.Dialer{
		TLSClientConfig:   tlsConfig,
		Proxy:             http.ProxyFromEnvironment,
		Subprotocols:      []string{"v4.channel.k8s.io"},
		EnableCompression: false,
	}

	header := http.Header{}
	header.Set("Authorization", "Bearer "+config.BearerToken)

	ctx := context.Background()

	slog.Debug("Ready to dial WebSocket", "url", u)
	conn, resp, err := dialer.DialContext(ctx, u.String(), header)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("WebSocket dial failed: %v\nResponse: %s", err, string(body))
		}
		return fmt.Errorf("WebSocket dial failed: %v", err)
	}
	defer conn.Close()

	slog.Debug("Connected to pod port-forward over WebSocket!")

	httpWrapper := NewHTTPServerWrapper(req.LocalPort, conn)
	httpWrapper.Start()
	fmt.Println("closing all ws")

	return nil
}

type HTTPServerWrapper struct {
	port   int
	wsConn *websocket.Conn
}

func NewHTTPServerWrapper(port int, conn *websocket.Conn) *HTTPServerWrapper {
	return &HTTPServerWrapper{
		port,
		conn,
	}
}

func (hw *HTTPServerWrapper) Start() {
	http.HandleFunc("/", hw.handleHTTP)

	slog.Debug("HTTP bridge listening", "port", hw.port)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", hw.port), nil))
}

func (hw *HTTPServerWrapper) handleHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO
	r.Header.Set("x-user-email", "test@danfoss.com")
	reqBytes, err := httpReqToBytes(r)
	if err != nil {
		log.Fatalf("error with http and bytes: %v", err)
	}

	msg := append([]byte{0}, reqBytes...) // first byte = channel index

	slog.Debug("Writing into WS")

	err = hw.wsConn.WriteMessage(websocket.BinaryMessage, msg)
	if err != nil {
		log.Fatalf("error writing into websocket: %v", err)
	}

	if err = hw.wsConn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		log.Fatalf("error setting deadline: %v", err)
	}

	slog.Debug("Reading back from WS")
	resp, err := readHTTPResp(hw.wsConn)
	if err != nil {
		log.Printf("error reading message: %v", err)
	}
	resp, err = readHTTPResp(hw.wsConn)
	if err != nil {
		log.Fatalf("error reading message: %v", err)
	}

	if len(resp) < 50 {
		slog.Debug("Writing back to HTTP", "resp", resp)
	} else {
		slog.Debug("Writing back to HTTP")
	}

	if _, err = w.Write(resp); err != nil {
		log.Fatalf("error writing http response: %v", err)
	}
}

func httpReqToBytes(req *http.Request) ([]byte, error) {
	var buf bytes.Buffer
	if err := req.Write(&buf); err != nil {
		return nil, fmt.Errorf("cannot marshal binary request")
	}
	return buf.Bytes(), nil
}

func readHTTPResp(conn *websocket.Conn) ([]byte, error) {
	buf := &bytes.Buffer{}

	for {
		_, frame, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) ||
				errors.Is(err, os.ErrDeadlineExceeded) {
				break
			}
			return nil, fmt.Errorf("error reading http response: %w", err)
		}

		if len(frame) == 0 {
			continue
		}

		channel := frame[0]
		if channel != 0 {
			slog.Debug("Non-data channel", "channel", channel, "payload", string(frame[1:]))
			continue
		}

		buf.Write(frame[1:])

		// Optional: detect end of HTTP response
		if bytes.Contains(buf.Bytes(), []byte("\r\n\r\n")) {
			// If Content-Length exists, you can check if we have the full body
			// Or if Transfer-Encoding: chunked, parse chunks
			// For HTTP/1.0 without keep-alive, end is when connection closes
			break
		}
	}

	return buf.Bytes(), nil
}
