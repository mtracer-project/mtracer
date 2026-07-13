package assertion

import (
	"fmt"

	"github.com/mtracer-project/mtracer/parser"
	"github.com/mtracer-project/mtracer/trace"

	"github.com/google/cel-go/cel"
)

type CelAssertion struct {
	name     string
	queries  []string
	programs []cel.Program
}

func (a *CelAssertion) Assert(t *trace.Trace) (bool, error) {
	if t == nil {
		return false, fmt.Errorf("trace is nil")
	}

	activation := map[string]any{
		"trace": t.ToProto(),
	}

	for i, program := range a.programs {
		out, _, err := program.Eval(activation)
		if err != nil {
			return false, err
		}

		result, ok := out.Value().(bool)
		if !ok {
			return false, fmt.Errorf("got %v, wanted bool result", out.Value())
		}

		if !result {
			return false, fmt.Errorf("assertion failed: %s (query: %s)", a.name, a.queries[i])
		}
	}

	return true, nil
}

func NewCelAssertion(dto *parser.AssertionDTO) (*CelAssertion, error) {
	queries := make([]string, 0, len(dto.Queries))
	for _, query := range dto.Queries {
		if q, ok := query.(string); ok {
			queries = append(queries, q)
		}
	}

	env, err := cel.NewEnv(
		cel.JSONFieldNames(true),
		cel.Types(&trace.TraceProto{}),
		cel.Variable(
			"trace",
			cel.ObjectType("mtracer.trace.TraceProto"),
		),
	)
	if err != nil {
		return nil, err
	}

	programs := make([]cel.Program, 0, len(queries))
	for _, query := range queries {
		ast, iss := env.Compile(query)
		if iss.Err() != nil {
			return nil, iss.Err()
		}
		if ast.OutputType() != cel.BoolType && ast.OutputType() != cel.DynType {
			return nil, fmt.Errorf("got %v, wanted bool result type", ast.OutputType())
		}
		program, err := env.Program(ast)
		if err != nil {
			return nil, err
		}
		programs = append(programs, program)
	}

	return &CelAssertion{
		name:     dto.Name,
		queries:  queries,
		programs: programs,
	}, nil
}
