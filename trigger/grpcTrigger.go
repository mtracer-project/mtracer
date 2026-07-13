package trigger

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	idgenerator "github.com/mtracer-project/mtracer/idGenerator"
	"github.com/mtracer-project/mtracer/parser"

	"github.com/bufbuild/protocompile"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

type GrpcTrigger struct {
	serverAddress     string
	service           string
	serviceDescriptor ServiceDescriptor
	method            string
	metadata          map[string][]string
	data              map[string]any

	baseDir     string
	idGenerator idgenerator.IdGenerator
	ctx         context.Context
}

func (g *GrpcTrigger) Init(dto *parser.TriggerDTO, idGenerator idgenerator.IdGenerator, baseDir string, ctx context.Context) error {
	if dto.Args == nil {
		return fmt.Errorf("invalid trigger arguments")
	}

	serverAddress, ok := dto.Args["serverAddress"].(string)
	if !ok {
		return fmt.Errorf("serverAddress argument is required and must be a string")
	}

	method, ok := dto.Args["method"].(string)
	if !ok {
		return fmt.Errorf("method argument is required and must be a string")
	}

	serviceName, methodName, err := ParseGrpcMethod(method)
	if err != nil {
		return fmt.Errorf("error while parsing method argument: %w", err)
	}

	metadata := make(map[string][]string)
	if md, ok := dto.Args["metadata"].(map[string]any); ok {
		for k, v := range md {
			if valStr, ok := v.(string); ok {
				metadata[k] = append(metadata[k], valStr)
			}
		}
	}

	var data map[string]any
	if d, ok := dto.Args["data"].(map[string]any); ok {
		data = d
	}

	serviceDescriptor, err := NewServiceDescriptor(dto, baseDir, serverAddress, ctx)
	if err != nil {
		return fmt.Errorf("error while creating service descriptor: %w", err)
	}

	g.serverAddress = serverAddress
	g.service = serviceName
	g.serviceDescriptor = serviceDescriptor
	g.method = methodName
	g.metadata = metadata
	g.data = data
	g.baseDir = baseDir
	g.idGenerator = idGenerator
	g.ctx = ctx
	return nil
}

func ParseGrpcMethod(methodName string) (string, string, error) {
	lastDot := strings.LastIndex(methodName, ".")
	if lastDot == -1 {
		return "", "", fmt.Errorf("invalid method format: %q. Expected 'Service.Method' or 'package.Service.Method'", methodName)
	}
	serviceName := methodName[:lastDot]
	methodOnly := methodName[lastDot+1:]
	return serviceName, methodOnly, nil
}

func (g *GrpcTrigger) Trigger() (TraceId, error) {
	traceId, err := g.idGenerator.Generate(idgenerator.TRACE_ID_LENGTH)
	if err != nil {
		return "", fmt.Errorf("error while generating trace ID: %w", err)
	}

	spanId, err := g.idGenerator.Generate(idgenerator.SPAN_ID_LENGTH)
	if err != nil {
		return "", fmt.Errorf("error while generating span ID: %w", err)
	}

	traceparent := getTraceparent(traceId, spanId)

	response, err := g.sendGRPCRequest(
		traceparent,
		g.serverAddress,
		g.service,
		g.method,
		g.metadata,
		g.data,
		g.ctx,
	)
	if err != nil {
		return "", fmt.Errorf("error while sending the gRPC request: %w", err)
	}

	slog.Info("gRPC Trigger Response", "response", response)

	traceIdObj, err := NewTraceId(traceId)
	if err != nil {
		return "", fmt.Errorf("error while creating TraceId object: %w", err)
	}

	return traceIdObj, nil
}

func (g *GrpcTrigger) sendGRPCRequest(traceparent, serverAddress, serviceName, methodName string, meta map[string][]string, data map[string]any, ctx context.Context) (string, error) {
	outCtx := getOutgoingContext(traceparent, meta, ctx)

	serviceDesc, err := g.serviceDescriptor.Get(serviceName)
	if err != nil {
		return "", fmt.Errorf("error while fetching service descriptor: %w", err)
	}
	if serviceDesc == nil {
		return "", fmt.Errorf("service '%s' not found in proto file", serviceName)
	}

	methodDesc := serviceDesc.Methods().ByName(protoreflect.Name(methodName))
	if methodDesc == nil {
		return "", fmt.Errorf("method '%s' not found in service '%s'", methodName, serviceName)
	}

	inputMsg := dynamicpb.NewMessage(methodDesc.Input())
	outputMsg := dynamicpb.NewMessage(methodDesc.Output())

	if err := populateInputMessage(inputMsg, data); err != nil {
		return "", fmt.Errorf("error while populating input message: %w", err)
	}

	conn, err := grpc.NewClient(serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", fmt.Errorf("failed to create gRPC client for %s: %w", serverAddress, err)
	}
	defer conn.Close() //nolint:errcheck

	fullMethod := fmt.Sprintf("/%s/%s", serviceDesc.FullName(), methodDesc.Name())

	return executeRequest(conn, fullMethod, inputMsg, outputMsg, outCtx)
}

func getOutgoingContext(traceparent string, meta map[string][]string, ctx context.Context) context.Context {
	md := metadata.New(nil)
	for k, values := range meta {
		for _, v := range values {
			md.Append(k, v)
		}
	}
	md.Append("traceparent", traceparent)
	return metadata.NewOutgoingContext(ctx, md)
}

func populateInputMessage(inputMsg *dynamicpb.Message, data map[string]any) error {
	if len(data) == 0 {
		return nil
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal input data: %w", err)
	}

	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
		Resolver:       protoregistry.GlobalTypes,
	}

	if err := unmarshaler.Unmarshal(jsonData, inputMsg); err != nil {
		return fmt.Errorf("failed to populate dynamic message via protojson: %w", err)
	}

	return nil
}

func executeRequest(conn *grpc.ClientConn, fullMethod string, inputMsg, outputMsg *dynamicpb.Message, outCtx context.Context) (string, error) {
	_ = conn.Invoke(outCtx, fullMethod, inputMsg, outputMsg) // nolint:errcheck

	marshaler := protojson.MarshalOptions{Multiline: true, EmitUnpopulated: true}
	respJson, err := marshaler.Marshal(outputMsg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response to JSON: %w", err)
	}
	return string(respJson), nil
}

type ServiceDescriptor interface {
	Get(serviceName string) (protoreflect.ServiceDescriptor, error)
}

func NewServiceDescriptor(dto *parser.TriggerDTO, baseDir, serverAddress string, ctx context.Context) (ServiceDescriptor, error) {
	if dto.Args == nil {
		return nil, fmt.Errorf("invalid trigger arguments")
	}

	descriptorSource, ok := dto.Args["descriptorSource"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("descriptorSource argument is required and must be an object")
	}

	sourceType, ok := descriptorSource["type"].(string)
	if !ok {
		return nil, fmt.Errorf("descriptorSource.type is required and must be a string")
	}

	switch strings.ToLower(sourceType) {
	case "file":
		return NewProtoFileServiceDescriptor(dto, baseDir, ctx)
	case "serverreflection":
		return NewServerReflectionServiceDescriptor(dto, serverAddress, ctx)
	default:
		return nil, fmt.Errorf("unsupported descriptor source type: %s", sourceType)
	}
}

type ProtoFileServiceDescriptor struct {
	protoPath string
	baseDir   string
	ctx       context.Context
}

func (p *ProtoFileServiceDescriptor) Get(serviceName string) (protoreflect.ServiceDescriptor, error) {
	importPaths := []string{".", p.baseDir}
	compilePath := p.protoPath

	if filepath.IsAbs(p.protoPath) {
		if rel, err := filepath.Rel(p.baseDir, p.protoPath); err == nil && !strings.HasPrefix(rel, "..") {
			compilePath = rel
		} else {
			importPaths = append(importPaths, filepath.Dir(p.protoPath))
			compilePath = filepath.Base(p.protoPath)
		}
	}

	resolver := &protocompile.SourceResolver{
		ImportPaths: importPaths,
	}
	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(resolver),
	}

	allFiles, err := compiler.Compile(p.ctx, compilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to compile proto file: %w", err)
	}

	if len(allFiles) == 0 {
		return nil, fmt.Errorf("no compiled files returned for %s", p.protoPath)
	}
	fileDesc := allFiles[0]

	var serviceDesc protoreflect.ServiceDescriptor
	for i := 0; i < fileDesc.Services().Len(); i++ {
		s := fileDesc.Services().Get(i)
		if string(s.Name()) == serviceName || string(s.FullName()) == serviceName {
			serviceDesc = s
			break
		}
	}
	return serviceDesc, nil
}

func NewProtoFileServiceDescriptor(dto *parser.TriggerDTO, baseDir string, ctx context.Context) (*ProtoFileServiceDescriptor, error) {
	if dto.Args == nil {
		return nil, fmt.Errorf("invalid trigger arguments")
	}

	descriptorSource, ok := dto.Args["descriptorSource"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("descriptorSource argument is required")
	}

	protoPath, ok := descriptorSource["protoPath"].(string)
	if !ok {
		return nil, fmt.Errorf("protoPath argument is required and must be a string")
	}
	resolvedProtoPath := resolvePath(baseDir, protoPath)

	return &ProtoFileServiceDescriptor{
		protoPath: resolvedProtoPath,
		baseDir:   baseDir,
		ctx:       ctx,
	}, nil
}

type ServerReflectionServiceDescriptor struct {
	serverAddress string
	ctx           context.Context
}

func (s *ServerReflectionServiceDescriptor) Get(serviceName string) (protoreflect.ServiceDescriptor, error) {
	conn, err := grpc.NewClient(s.serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("impossible to connect to [%s]: %w", s.serverAddress, err)
	}
	defer conn.Close() //nolint:errcheck

	refClient := grpcreflect.NewClientAuto(s.ctx, conn)
	defer refClient.Reset()

	serviceDesc, err := refClient.ResolveService(serviceName)
	if err != nil {
		return nil, fmt.Errorf("reflection failed for service '%s' (does the server have reflection enabled?): %w", serviceName, err)
	}

	return serviceDesc.UnwrapService(), nil
}

func NewServerReflectionServiceDescriptor(dto *parser.TriggerDTO, serverAddress string, ctx context.Context) (*ServerReflectionServiceDescriptor, error) {
	return &ServerReflectionServiceDescriptor{
		serverAddress: serverAddress,
		ctx:           ctx,
	}, nil
}

func (t *GrpcTrigger) Example() string {
	return `trigger:
  type: "gRPC"
  args:
    serverAddress: "localhost:50051"
    descriptorSource:
      type: "file" # file | serverReflection
      protoPath: "path/to/your/service.proto"
    method: "package.service.method"
    metadata:
      key1: "value1"
      key2: "value2"
    data:
      field1: "value1"
      field2: 123
      field3: "20s"
      field4: "2023-10-01T12:00:00Z"`
}
