package trigger

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	idgenerator "github.com/mtracer-project/mtracer/idGenerator"
	"github.com/mtracer-project/mtracer/parser"

	nats "github.com/nats-io/nats.go"
	nkeys "github.com/nats-io/nkeys"
)

const (
	NATS_SERVER_URL_FORMAT = "nats://%s"
)

type NATSConnector interface {
	Connect(serverAddress string, caPemPath *string) (*nats.Conn, error)
}

type NATSTrigger struct {
	serverAddress string
	subject       string
	headers       map[string][]string
	data          []byte
	natsConn      NATSConnector
	caPemPath     *string

	baseDir     string
	idGenerator idgenerator.IdGenerator
}

func (t *NATSTrigger) Init(dto *parser.TriggerDTO, idGenerator idgenerator.IdGenerator, baseDir string, ctx context.Context) error {
	if dto.Args == nil {
		return fmt.Errorf("invalid trigger arguments")
	}

	serverAddress, ok := dto.Args["serverAddress"].(string)
	if !ok {
		return fmt.Errorf("serverAddress argument is required and must be a string")
	}

	subject, ok := dto.Args["subject"].(string)
	if !ok {
		return fmt.Errorf("subject argument is required and must be a string")
	}

	headers := make(map[string][]string)
	if hdrs, ok := dto.Args["headers"].(map[string]any); ok {
		for k, v := range hdrs {
			if valStr, ok := v.(string); ok {
				headers[k] = append(headers[k], valStr)
			}
		}
	}

	var caPemPath *string
	if path, ok := dto.Args["caPemPath"].(string); ok {
		resolvedPath := resolvePath(baseDir, path)
		caPemPath = &resolvedPath
	}

	var data []byte
	if d, ok := dto.Args["data"].(string); ok {
		data = []byte(d)
	}

	connector, err := NewNATSConnector(dto, baseDir)
	if err != nil {
		return fmt.Errorf("error creating NATS connector: %w", err)
	}

	t.serverAddress = serverAddress
	t.subject = subject
	t.headers = headers
	t.data = data
	t.natsConn = connector
	t.idGenerator = idGenerator
	t.baseDir = baseDir
	t.caPemPath = caPemPath
	return nil
}

func (t *NATSTrigger) Trigger() (TraceId, error) {
	traceId, err := t.idGenerator.Generate(idgenerator.TRACE_ID_LENGTH)
	if err != nil {
		return "", fmt.Errorf("error while generating trace ID: %w", err)
	}

	spanId, err := t.idGenerator.Generate(idgenerator.SPAN_ID_LENGTH)
	if err != nil {
		return "", fmt.Errorf("error while generating span ID: %w", err)
	}

	nc, err := t.natsConn.Connect(t.serverAddress, t.caPemPath)
	if err != nil {
		return "", fmt.Errorf("error while connecting to NATS: %w", err)
	}
	defer nc.Close() //nolint:errcheck

	msg := nats.NewMsg(t.subject)
	msg.Data = t.data
	if msg.Header == nil {
		msg.Header = nats.Header{}
	}

	for key, values := range t.headers {
		for _, value := range values {
			msg.Header.Add(key, value)
		}
	}

	traceparent := getTraceparent(traceId, spanId)
	msg.Header.Set("traceparent", traceparent)

	slog.Info("NATS Trigger", "subject", t.subject, "headers", msg.Header, "traceparent", traceparent, "data", string(t.data))

	if err := nc.PublishMsg(msg); err != nil {
		return "", fmt.Errorf("error while publishing NATS message: %w", err)
	}

	if err := nc.Flush(); err != nil {
		return "", fmt.Errorf("error while flushing NATS message: %w", err)
	}

	traceIdObj, err := NewTraceId(traceId)
	if err != nil {
		return "", fmt.Errorf("error while creating TraceId object: %w", err)
	}

	return traceIdObj, nil
}

func NewNATSConnector(dto *parser.TriggerDTO, baseDir string) (NATSConnector, error) {
	authType, ok := dto.Args["authType"].(string)
	if !ok {
		return nil, fmt.Errorf("authType argument is required and must be a string")
	}

	switch strings.ToLower(authType) {
	case "noauth":
		return NewNATSNoAuthConnector(), nil
	case "userpass":
		username, ok := dto.Args["username"].(string)
		if !ok {
			return nil, fmt.Errorf("username argument is required and must be a string")
		}
		password, ok := dto.Args["password"].(string)
		if !ok {
			return nil, fmt.Errorf("password argument is required and must be a string")
		}
		return NewNATSUserPassConnector(username, password), nil
	case "token":
		token, ok := dto.Args["token"].(string)
		if !ok {
			return nil, fmt.Errorf("token argument is required and must be a string")
		}
		return NewNATSTokenConnector(token), nil
	case "jwt":
		jwt, ok := dto.Args["jwt"].(string)
		if !ok {
			return nil, fmt.Errorf("jwt argument is required and must be a string")
		}
		seed, ok := dto.Args["seed"].(string)
		if !ok {
			return nil, fmt.Errorf("seed argument is required and must be a string")
		}
		return NewNATSJWTConnector(jwt, seed), nil
	case "nkey":
		seed, ok := dto.Args["seed"].(string)
		if !ok {
			return nil, fmt.Errorf("seed argument is required and must be a string")
		}
		return NewNATSNKeyConnector(seed), nil
	case "mtls":
		clientCertPath, ok := dto.Args["clientCertPath"].(string)
		if !ok {
			return nil, fmt.Errorf("clientCertPath argument is required and must be a string")
		}
		clientKeyPath, ok := dto.Args["clientKeyPath"].(string)
		if !ok {
			return nil, fmt.Errorf("clientKeyPath argument is required and must be a string")
		}
		return NewNATSMTLSConnector(clientCertPath, clientKeyPath, baseDir), nil
	default:
		return nil, fmt.Errorf("unsupported NATS authentication type: %s", authType)
	}
}

// No authentication
type NATSNoAuthConnector struct{}

func (c *NATSNoAuthConnector) Connect(serverAddress string, caPemPath *string) (*nats.Conn, error) {
	return connectNATS(serverAddress, caPemPath)
}

func NewNATSNoAuthConnector() *NATSNoAuthConnector {
	return &NATSNoAuthConnector{}
}

// User and password authentication
type NATSUserPassConnector struct {
	username string
	password string
}

func (c *NATSUserPassConnector) Connect(serverAddress string, caPemPath *string) (*nats.Conn, error) {
	return connectNATS(serverAddress, caPemPath, nats.UserInfo(c.username, c.password))
}

func NewNATSUserPassConnector(username, password string) *NATSUserPassConnector {
	return &NATSUserPassConnector{
		username: username,
		password: password,
	}
}

// Token authentication
type NATSTokenConnector struct {
	token string
}

func (c *NATSTokenConnector) Connect(serverAddress string, caPemPath *string) (*nats.Conn, error) {
	return connectNATS(serverAddress, caPemPath, nats.Token(c.token))
}

func NewNATSTokenConnector(token string) *NATSTokenConnector {
	return &NATSTokenConnector{
		token: token,
	}
}

// JWT authentication
type NATSJWTConnector struct {
	jwt  string
	seed string
}

func (c *NATSJWTConnector) Connect(serverAddress string, caPemPath *string) (*nats.Conn, error) {
	return connectNATS(serverAddress, caPemPath, nats.UserJWTAndSeed(c.jwt, c.seed))
}

func NewNATSJWTConnector(jwt string, seed string) *NATSJWTConnector {
	return &NATSJWTConnector{
		jwt:  jwt,
		seed: seed,
	}
}

// NKey authentication
type NATSNKeyConnector struct {
	seed string
}

func (c *NATSNKeyConnector) Connect(serverAddress string, caPemPath *string) (*nats.Conn, error) {
	kp, err := nkeys.FromSeed([]byte(c.seed))
	if err != nil {
		return nil, fmt.Errorf("invalid NATS nkey seed: %w", err)
	}
	defer kp.Wipe()

	publicKey, err := kp.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("error deriving NATS public key from seed: %w", err)
	}

	signatureCallback := func(nonce []byte) ([]byte, error) {
		return kp.Sign(nonce)
	}

	return connectNATS(serverAddress, caPemPath, nats.Nkey(publicKey, signatureCallback))
}

func NewNATSNKeyConnector(seed string) *NATSNKeyConnector {
	return &NATSNKeyConnector{
		seed: seed,
	}
}

// mTLS authentication
type NATSMTLSConnector struct {
	clientCertPath string
	clientKeyPath  string
	baseDir        string
}

func (c *NATSMTLSConnector) Connect(serverAddress string, caPemPath *string) (*nats.Conn, error) {
	clientCertPath := resolvePath(c.baseDir, c.clientCertPath)
	clientKeyPath := resolvePath(c.baseDir, c.clientKeyPath)
	options := []nats.Option{nats.ClientCert(clientCertPath, clientKeyPath)}
	return connectNATS(serverAddress, caPemPath, options...)
}

func NewNATSMTLSConnector(clientCertPath, clientKeyPath, baseDir string) *NATSMTLSConnector {
	return &NATSMTLSConnector{
		clientCertPath: clientCertPath,
		clientKeyPath:  clientKeyPath,
		baseDir:        baseDir,
	}
}

// Helper functions to connect to NATS and resolve paths if provided
func connectNATS(serverAddress string, caPemPath *string, options ...nats.Option) (*nats.Conn, error) {
	if caPemPath != nil {
		options = append(options, nats.RootCAs(*caPemPath))
	}

	nc, err := nats.Connect(fmt.Sprintf(NATS_SERVER_URL_FORMAT, serverAddress), options...)
	if err != nil {
		return nil, err
	}

	return nc, nil
}

func (t *NATSTrigger) Example() string {
	return `trigger:
  type: "nats"
  args:
    serverAddress: "localhost:4222"
    subject: "test.subject"
    headers:
      - Content-Type: "application/json"
    data: '{"key": "value"}'
    caPemPath: "path/to/ca.pem"
    authType: "userpass" # noauth | userpass | token | jwt | nkey | mtls
    username: "nats-user" # required if authType is userpass
    password: "nats-password" # required if authType is userpass`
}
