// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component/componenttest"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottltest"
)

func Test_NewFunctionCall_invalid(t *testing.T) {
	functions := CreateFactoryMap(
		createFactory(
			"testing_error",
			&errorFunctionArguments{},
			functionThatHasAnError,
		),
		createFactory[any](
			"testing_getsetter",
			&getSetterArguments{},
			functionWithGetSetter,
		),
		createFactory[any](
			"testing_getter",
			&getterArguments{},
			functionWithGetter,
		),
		createFactory[any](
			"testing_multiple_args",
			&multipleArgsArguments{},
			functionWithMultipleArgs,
		),
		createFactory[any](
			"testing_string",
			&stringArguments{},
			functionWithString,
		),
		createFactory(
			"testing_string_slice",
			&stringSliceArguments{},
			functionWithStringSlice,
		),
		createFactory(
			"testing_byte_slice",
			&byteSliceArguments{},
			functionWithByteSlice,
		),
		createFactory[any](
			"testing_enum",
			&enumArguments{},
			functionWithEnum,
		),
		createFactory(
			"non_pointer",
			errorFunctionArguments{},
			functionThatHasAnError,
		),
		createFactory(
			"testing_unknown_function",
			&functionGetterArguments{},
			functionWithFunctionGetter,
		),
		createFactory[any](
			"testing_functiongetter",
			&functionGetterArguments{},
			functionWithFunctionGetter,
		),
	)

	p, _ := NewParser(
		functions,
		testParsePath,
		componenttest.NewNopTelemetrySettings(),
		WithEnumParser[any](testParseEnum),
	)

	tests := []struct {
		name string
		inv  editor
	}{
		{
			name: "unknown function",
			inv: editor{
				Function:  "unknownfunc",
				Arguments: []value{},
			},
		},
		{
			name: "Invalid Function Name",
			inv: editor{
				Function: "testing_functiongetter",
				Arguments: []value{
					{
						String: (ottltest.Strp("SHA256")),
					},
				},
			},
		},
		{
			name: "not accessor",
			inv: editor{
				Function: "testing_getsetter",
				Arguments: []value{
					{
						String: ottltest.Strp("not path"),
					},
				},
			},
		},
		{
			name: "not reader (invalid function)",
			inv: editor{
				Function: "testing_getter",
				Arguments: []value{
					{
						Literal: &mathExprLiteral{
							Converter: &converter{
								Function: "Unknownfunc",
							},
						},
					},
				},
			},
		},
		{
			name: "not enough args",
			inv: editor{
				Function: "testing_multiple_args",
				Arguments: []value{
					{
						Literal: &mathExprLiteral{
							Path: &Path{
								Fields: []Field{
									{
										Name: "name",
									},
								},
							},
						},
					},
					{
						String: ottltest.Strp("test"),
					},
				},
			},
		},
		{
			name: "too many args",
			inv: editor{
				Function: "testing_multiple_args",
				Arguments: []value{
					{
						Literal: &mathExprLiteral{
							Path: &Path{
								Fields: []Field{
									{
										Name: "name",
									},
								},
							},
						},
					},
					{
						String: ottltest.Strp("test"),
					},
					{
						String: ottltest.Strp("test"),
					},
				},
			},
		},
		{
			name: "not enough args with telemetrySettings",
			inv: editor{
				Function: "testing_telemetry_settings_first",
				Arguments: []value{
					{
						String: ottltest.Strp("test"),
					},
					{
						List: &list{
							Values: []value{
								{
									String: ottltest.Strp("test"),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "too many args with telemetrySettings",
			inv: editor{
				Function: "testing_telemetry_settings_first",
				Arguments: []value{
					{
						String: ottltest.Strp("test"),
					},
					{
						List: &list{
							Values: []value{
								{
									String: ottltest.Strp("test"),
								},
							},
						},
					},
					{
						Literal: &mathExprLiteral{
							Int: ottltest.Intp(10),
						},
					},
					{
						Literal: &mathExprLiteral{
							Int: ottltest.Intp(10),
						},
					},
				},
			},
		},
		{
			name: "not matching arg type",
			inv: editor{
				Function: "testing_string",
				Arguments: []value{
					{
						Literal: &mathExprLiteral{
							Int: ottltest.Intp(10),
						},
					},
				},
			},
		},
		{
			name: "not matching arg type when byte slice",
			inv: editor{
				Function: "testing_byte_slice",
				Arguments: []value{
					{
						String: ottltest.Strp("test"),
					},
					{
						String: ottltest.Strp("test"),
					},
					{
						String: ottltest.Strp("test"),
					},
				},
			},
		},
		{
			name: "mismatching slice element type",
			inv: editor{
				Function: "testing_string_slice",
				Arguments: []value{
					{
						List: &list{
							Values: []value{
								{
									String: ottltest.Strp("test"),
								},
								{
									Literal: &mathExprLiteral{
										Int: ottltest.Intp(10),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "mismatching slice argument type",
			inv: editor{
				Function: "testing_string_slice",
				Arguments: []value{
					{
						String: ottltest.Strp("test"),
					},
				},
			},
		},
		{
			name: "function call returns error",
			inv: editor{
				Function: "testing_error",
			},
		},
		{
			name: "Enum not found",
			inv: editor{
				Function: "testing_enum",
				Arguments: []value{
					{
						Enum: (*EnumSymbol)(ottltest.Strp("SYMBOL_NOT_FOUND")),
					},
				},
			},
		},
		{
			name: "Unknown Function",
			inv: editor{
				Function: "testing_functiongetter",
				Arguments: []value{
					{
						FunctionName: (ottltest.Strp("SHA256")),
					},
				},
			},
		},
		{
			name: "factory definition uses a non-pointer Arguments value",
			inv: editor{
				Function: "non_pointer",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.newFunctionCall(tt.inv)
			t.Log(err)
			assert.Error(t, err)
		})
	}
}

func Test_NewFunctionCall(t *testing.T) {
	p, _ := NewParser(
		defaultFunctionsForTests(),
		testParsePath,
		componenttest.NewNopTelemetrySettings(),
		WithEnumParser[any](testParseEnum),
	)

	tests := []struct {
		name string
		inv  editor
		want any
	}{
		{
			name: "no arguments",
			inv: editor{
				Function: "testing_noop",
				Arguments: []value{
					{
						List: &list{
							Values: []value{},
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "empty slice arg",
			inv: editor{
				Function: "testing_string_slice",
				Arguments: []value{
					{
						List: &list{
							Values: []value{},
						},
					},
				},
			},
			want: 0,
		},
		{
			name: "string slice arg",
			inv: editor{
				Function: "testing_string_slice",
				Arguments: []value{
					{
						List: &list{
							Values: []value{
								{
									String: ottltest.Strp("test"),
								},
								{
									String: ottltest.Strp("test"),
								},
								{
									String: ottltest.Strp("test"),
								},
							},
						},
					},
				},
			},
			want: 3,
		},
		{
			name: "float slice arg",
			inv: editor{
				Function: "testing_float_slice",
				Arguments: []value{
					{
						List: &list{
							Values: []value{
								{
									Literal: &mathExprLiteral{
										Float: ottltest.Floatp(1.1),
									},
								},
								{
									Literal: &mathExprLiteral{
										Float: ottltest.Floatp(1.2),
									},
								},
								{
									Literal: &mathExprLiteral{
										Float: ottltest.Floatp(1.3),
									},
								},
							},
						},
					},
				},
			},
			want: 3,
		},
		{
			name: "int slice arg",
			inv: editor{
				Function: "testing_int_slice",
				Arguments: []value{
					{
						List: &list{
							Values: []value{
								{
									Literal: &mathExprLiteral{
										Int: ottltest.Intp(1),
									},
								},
								{
									Literal: &mathExprLiteral{
										Int: ottltest.Intp(1),
									},
								},
								{
									Literal: &mathExprLiteral{
										Int: ottltest.Intp(1),
									},
								},
							},
						},
					},
				},
			},
			want: 3,
		},
		{
			name: "getter slice arg",
			inv: editor{
				Function: "testing_getter_slice",
				Arguments: []value{
					{
						List: &list{
							Values: []value{
								{
									Literal: &mathExprLiteral{
										Path: &Path{
											Fields: []Field{
												{
													Name: "name",
												},
											},
										},
									},
								},
								{
									String: ottltest.Strp("test"),
								},
								{
									Literal: &mathExprLiteral{
										Int: ottltest.Intp(1),
									},
								},
								{
									Literal: &mathExprLiteral{
										Float: ottltest.Floatp(1.1),
									},
								},
								{
									Bool: (*boolean)(ottltest.Boolp(true)),
								},
								{
									Enum: (*EnumSymbol)(ottltest.Strp("TEST_ENUM")),
								},
								{
									List: &list{
										Values: []value{
											{
												String: ottltest.Strp("test"),
											},
											{
												String: ottltest.Strp("test"),
											},
										},
									},
								},
								{
									List: &list{
										Values: []value{
											{
												String: ottltest.Strp("test"),
											},
											{
												List: &list{
													Values: []value{
														{
															String: ottltest.Strp("test"),
														},
														{
															List: &list{
																Values: []value{
																	{
																		String: ottltest.Strp("test"),
																	},
																	{
																		String: ottltest.Strp("test"),
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
								{
									Literal: &mathExprLiteral{
										Converter: &converter{
											Function: "testing_getter",
											Arguments: []value{
												{
													Literal: &mathExprLiteral{
														Path: &Path{
															Fields: []Field{
																{
																	Name: "name",
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: 9,
		},
		{
			name: "stringgetter slice arg",
			inv: editor{
				Function: "testing_stringgetter_slice",
				Arguments: []value{
					{
						List: &list{
							Values: []value{
								{
									String: ottltest.Strp("test"),
								},
								{
									String: ottltest.Strp("also test"),
								},
							},
						},
					},
				},
			},
			want: 2,
		},
		{
			name: "durationgetter slice arg",
			inv: editor{
				Function: "testing_durationgetter_slice",
				Arguments: []value{
					{
						List: &list{
							Values: []value{
								{
									String: ottltest.Strp("test"),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "timegetter slice arg",
			inv: editor{
				Function: "testing_timegetter_slice",
				Arguments: []value{
					{
						List: &list{
							Values: []value{
								{
									String: ottltest.Strp("test"),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "floatgetter slice arg",
			inv: editor{
				Function: "testing_floatgetter_slice",
				Arguments: []value{
					{
						List: &list{
							Values: []value{
								{
									String: ottltest.Strp("1.1"),
								},
								{
									Literal: &mathExprLiteral{
										Float: ottltest.Floatp(1),
									},
								},
							},
						},
					},
				},
			},
			want: 2,
		},
		{
			name: "intgetter slice arg",
			inv: editor{
				Function: "testing_intgetter_slice",
				Arguments: []value{
					{
						List: &list{
							Values: []value{
								{
									Literal: &mathExprLiteral{
										Int: ottltest.Intp(1),
									},
								},
								{
									Literal: &mathExprLiteral{
										Int: ottltest.Intp(2),
									},
								},
							},
						},
					},
				},
			},
			want: 2,
		},
		{
			name: "pmapgetter slice arg",
			inv: editor{
				Function: "testing_pmapgetter_slice",
				Arguments: []value{
					{
						List: &list{
							Values: []value{
								{
									Literal: &mathExprLiteral{
										Path: &Path{
											Fields: []Field{
												{
													Name: "name",
												},
											},
										},
									},
								},
								{
									Literal: &mathExprLiteral{
										Path: &Path{
											Fields: []Field{
												{
													Name: "name",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: 2,
		},
		{
			name: "stringlikegetter slice arg",
			inv: editor{
				Function: "testing_stringlikegetter_slice",
				Arguments: []value{
					{
						List: &list{
							Values: []value{
								{
									String: ottltest.Strp("test"),
								},
								{
									Literal: &mathExprLiteral{
										Int: ottltest.Intp(1),
									},
								},
							},
						},
					},
				},
			},
			want: 2,
		},
		{
			name: "floatlikegetter slice arg",
			inv: editor{
				Function: "testing_floatlikegetter_slice",
				Arguments: []value{
					{
						List: &list{
							Values: []value{
								{
									String: ottltest.Strp("1.1"),
								},
								{
									Literal: &mathExprLiteral{
										Float: ottltest.Floatp(1.1),
									},
								},
							},
						},
					},
				},
			},
			want: 2,
		},
		{
			name: "intlikegetter slice arg",
			inv: editor{
				Function: "testing_intlikegetter_slice",
				Arguments: []value{
					{
						List: &list{
							Values: []value{
								{
									String: ottltest.Strp("1"),
								},
								{
									Literal: &mathExprLiteral{
										Float: ottltest.Floatp(1.1),
									},
								},
							},
						},
					},
				},
			},
			want: 2,
		},
		{
			name: "setter arg",
			inv: editor{
				Function: "testing_setter",
				Arguments: []value{
					{
						Literal: &mathExprLiteral{
							Path: &Path{
								Fields: []Field{
									{
										Name: "name",
									},
								},
							},
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "getsetter arg",
			inv: editor{
				Function: "testing_getsetter",
				Arguments: []value{
					{
						Literal: &mathExprLiteral{
							Path: &Path{
								Fields: []Field{
									{
										Name: "name",
									},
								},
							},
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "getter arg",
			inv: editor{
				Function: "testing_getter",
				Arguments: []value{
					{
						Literal: &mathExprLiteral{
							Path: &Path{
								Fields: []Field{
									{
										Name: "name",
									},
								},
							},
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "getter arg with nil literal",
			inv: editor{
				Function: "testing_getter",
				Arguments: []value{
					{
						IsNil: (*isNil)(ottltest.Boolp(true)),
					},
				},
			},
			want: nil,
		},
		{
			name: "getter arg with list",
			inv: editor{
				Function: "testing_getter",
				Arguments: []value{
					{
						List: &list{
							Values: []value{
								{
									String: ottltest.Strp("test"),
								},
								{
									Literal: &mathExprLiteral{
										Int: ottltest.Intp(1),
									},
								},
								{
									Literal: &mathExprLiteral{
										Float: ottltest.Floatp(1.1),
									},
								},
								{
									Bool: (*boolean)(ottltest.Boolp(true)),
								},
								{
									Bytes: (*byteSlice)(&[]byte{1, 2, 3, 4, 5, 6, 7, 8}),
								},
								{
									Literal: &mathExprLiteral{
										Path: &Path{
											Fields: []Field{
												{
													Name: "name",
												},
											},
										},
									},
								},
								{
									Literal: &mathExprLiteral{
										Converter: &converter{
											Function: "testing_getter",
											Arguments: []value{
												{
													Literal: &mathExprLiteral{
														Path: &Path{
															Fields: []Field{
																{
																	Name: "name",
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "stringgetter arg",
			inv: editor{
				Function: "testing_stringgetter",
				Arguments: []value{
					{
						String: ottltest.Strp("test"),
					},
				},
			},
			want: nil,
		},
		{
			name: "durationgetter arg",
			inv: editor{
				Function: "testing_durationgetter",
				Arguments: []value{
					{
						String: ottltest.Strp("test"),
					},
				},
			},
		},
		{
			name: "timegetter arg",
			inv: editor{
				Function: "testing_timegetter",
				Arguments: []value{
					{
						String: ottltest.Strp("test"),
					},
				},
			},
		},
		{
			name: "functiongetter arg (Uppercase)",
			inv: editor{
				Function: "testing_functiongetter",
				Arguments: []value{
					{
						FunctionName: (ottltest.Strp("SHA256")),
					},
				},
			},
			want: "hashstring",
		},
		{
			name: "functiongetter arg",
			inv: editor{
				Function: "testing_functiongetter",
				Arguments: []value{
					{
						FunctionName: (ottltest.Strp("Sha256")),
					},
				},
			},
			want: "hashstring",
		},
		{
			name: "stringlikegetter arg",
			inv: editor{
				Function: "testing_stringlikegetter",
				Arguments: []value{
					{
						Bool: (*boolean)(ottltest.Boolp(false)),
					},
				},
			},
			want: nil,
		},
		{
			name: "floatgetter arg",
			inv: editor{
				Function: "testing_floatgetter",
				Arguments: []value{
					{
						String: ottltest.Strp("1.1"),
					},
				},
			},
			want: nil,
		},
		{
			name: "floatlikegetter arg",
			inv: editor{
				Function: "testing_floatlikegetter",
				Arguments: []value{
					{
						Bool: (*boolean)(ottltest.Boolp(false)),
					},
				},
			},
			want: nil,
		},
		{
			name: "intgetter arg",
			inv: editor{
				Function: "testing_intgetter",
				Arguments: []value{
					{
						Literal: &mathExprLiteral{
							Int: ottltest.Intp(1),
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "intlikegetter arg",
			inv: editor{
				Function: "testing_intgetter",
				Arguments: []value{
					{
						Literal: &mathExprLiteral{
							Float: ottltest.Floatp(1.1),
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "pmapgetter arg",
			inv: editor{
				Function: "testing_pmapgetter",
				Arguments: []value{
					{
						Literal: &mathExprLiteral{
							Path: &Path{
								Fields: []Field{
									{
										Name: "name",
									},
								},
							},
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "string arg",
			inv: editor{
				Function: "testing_string",
				Arguments: []value{
					{
						String: ottltest.Strp("test"),
					},
				},
			},
			want: nil,
		},
		{
			name: "float arg",
			inv: editor{
				Function: "testing_float",
				Arguments: []value{
					{
						Literal: &mathExprLiteral{
							Float: ottltest.Floatp(1.1),
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "int arg",
			inv: editor{
				Function: "testing_int",
				Arguments: []value{
					{
						Literal: &mathExprLiteral{
							Int: ottltest.Intp(1),
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "bool arg",
			inv: editor{
				Function: "testing_bool",
				Arguments: []value{
					{
						Bool: (*boolean)(ottltest.Boolp(true)),
					},
				},
			},
			want: nil,
		},
		{
			name: "byteSlice arg",
			inv: editor{
				Function: "testing_byte_slice",
				Arguments: []value{
					{
						Bytes: (*byteSlice)(&[]byte{1, 2, 3, 4, 5, 6, 7, 8}),
					},
				},
			},
			want: nil,
		},
		{
			name: "multiple args",
			inv: editor{
				Function: "testing_multiple_args",
				Arguments: []value{
					{
						Literal: &mathExprLiteral{
							Path: &Path{
								Fields: []Field{
									{
										Name: "name",
									},
								},
							},
						},
					},
					{
						String: ottltest.Strp("test"),
					},
					{
						Literal: &mathExprLiteral{
							Float: ottltest.Floatp(1.1),
						},
					},
					{
						Literal: &mathExprLiteral{
							Int: ottltest.Intp(1),
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "Enum arg",
			inv: editor{
				Function: "testing_enum",
				Arguments: []value{
					{
						Enum: (*EnumSymbol)(ottltest.Strp("TEST_ENUM")),
					},
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, err := p.newFunctionCall(tt.inv)
			assert.NoError(t, err)

			if tt.want != nil {
				result, _ := fn.Eval(context.Background(), nil)
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func functionWithNoArguments() (ExprFunc[any], error) {
	return func(context.Context, any) (any, error) {
		return nil, nil
	}, nil
}

func functionWithErr() (ExprFunc[any], error) {
	return func(context.Context, any) (any, error) {
		return nil, nil
	}, fmt.Errorf("error")
}

type stringSliceArguments struct {
	Strings []string
}

func functionWithStringSlice(strs []string) (ExprFunc[any], error) {
	return func(context.Context, any) (any, error) {
		return len(strs), nil
	}, nil
}

type floatSliceArguments struct {
	Floats []float64
}

func functionWithFloatSlice(floats []float64) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return len(floats), nil
	}, nil
}

type intSliceArguments struct {
	Ints []int64
}

func functionWithIntSlice(ints []int64) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return len(ints), nil
	}, nil
}

type byteSliceArguments struct {
	Bytes []byte
}

func functionWithByteSlice(bytes []byte) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return len(bytes), nil
	}, nil
}

type getterSliceArguments struct {
	Getters []Getter[any]
}

func functionWithGetterSlice(getters []Getter[interface{}]) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return len(getters), nil
	}, nil
}

type stringGetterSliceArguments struct {
	StringGetters []StringGetter[any]
}

func functionWithStringGetterSlice(getters []StringGetter[interface{}]) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return len(getters), nil
	}, nil
}

type durationGetterSliceArguments struct {
	DurationGetters []DurationGetter[any]
}

func functionWithDurationGetterSlice(_ []DurationGetter[interface{}]) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return nil, nil
	}, nil
}

type timeGetterSliceArguments struct {
	TimeGetters []TimeGetter[any]
}

func functionWithTimeGetterSlice(_ []TimeGetter[interface{}]) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return nil, nil
	}, nil
}

type floatGetterSliceArguments struct {
	FloatGetters []FloatGetter[any]
}

func functionWithFloatGetterSlice(getters []FloatGetter[interface{}]) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return len(getters), nil
	}, nil
}

type intGetterSliceArguments struct {
	IntGetters []IntGetter[any]
}

func functionWithIntGetterSlice(getters []IntGetter[interface{}]) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return len(getters), nil
	}, nil
}

type pMapGetterSliceArguments struct {
	PMapGetters []PMapGetter[any]
}

func functionWithPMapGetterSlice(getters []PMapGetter[interface{}]) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return len(getters), nil
	}, nil
}

type stringLikeGetterSliceArguments struct {
	StringLikeGetters []StringLikeGetter[any]
}

func functionWithStringLikeGetterSlice(getters []StringLikeGetter[interface{}]) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return len(getters), nil
	}, nil
}

type floatLikeGetterSliceArguments struct {
	FloatLikeGetters []FloatLikeGetter[any]
}

func functionWithFloatLikeGetterSlice(getters []FloatLikeGetter[interface{}]) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return len(getters), nil
	}, nil
}

type intLikeGetterSliceArguments struct {
	IntLikeGetters []IntLikeGetter[any]
}

func functionWithIntLikeGetterSlice(getters []IntLikeGetter[interface{}]) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return len(getters), nil
	}, nil
}

type setterArguments struct {
	SetterArg Setter[any]
}

func functionWithSetter(Setter[interface{}]) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return "anything", nil
	}, nil
}

type getSetterArguments struct {
	GetSetterArg GetSetter[any]
}

func functionWithGetSetter(GetSetter[interface{}]) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return "anything", nil
	}, nil
}

type getterArguments struct {
	GetterArg Getter[any]
}

func functionWithGetter(Getter[interface{}]) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return "anything", nil
	}, nil
}

type stringGetterArguments struct {
	StringGetterArg StringGetter[any]
}

func functionWithStringGetter(StringGetter[interface{}]) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return "anything", nil
	}, nil
}

type durationGetterArguments struct {
	DurationGetterArg DurationGetter[any]
}

func functionWithDurationGetter(DurationGetter[interface{}]) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return "anything", nil
	}, nil
}

type timeGetterArguments struct {
	TimeGetterArg TimeGetter[any]
}

func functionWithTimeGetter(TimeGetter[interface{}]) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return "anything", nil
	}, nil
}

type functionGetterArguments struct {
	FunctionGetterArg FunctionGetter[any]
}

func functionWithFunctionGetter(FunctionGetter[interface{}]) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return "hashstring", nil
	}, nil
}

type stringLikeGetterArguments struct {
	StringLikeGetterArg StringLikeGetter[any]
}

func functionWithStringLikeGetter(StringLikeGetter[interface{}]) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return "anything", nil
	}, nil
}

type floatGetterArguments struct {
	FloatGetterArg FloatGetter[any]
}

func functionWithFloatGetter(FloatGetter[interface{}]) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return "anything", nil
	}, nil
}

type floatLikeGetterArguments struct {
	FloatLikeGetterArg FloatLikeGetter[any]
}

func functionWithFloatLikeGetter(FloatLikeGetter[interface{}]) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return "anything", nil
	}, nil
}

type intGetterArguments struct {
	IntGetterArg IntGetter[any]
}

func functionWithIntGetter(IntGetter[interface{}]) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return "anything", nil
	}, nil
}

type intLikeGetterArguments struct {
	IntLikeGetterArg IntLikeGetter[any]
}

func functionWithIntLikeGetter(IntLikeGetter[interface{}]) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return "anything", nil
	}, nil
}

type pMapGetterArguments struct {
	PMapArg PMapGetter[any]
}

func functionWithPMapGetter(PMapGetter[interface{}]) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return "anything", nil
	}, nil
}

type stringArguments struct {
	StringArg string
}

func functionWithString(string) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return "anything", nil
	}, nil
}

type floatArguments struct {
	FloatArg float64
}

func functionWithFloat(float64) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return "anything", nil
	}, nil
}

type intArguments struct {
	IntArg int64
}

func functionWithInt(int64) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return "anything", nil
	}, nil
}

type boolArguments struct {
	BoolArg bool
}

func functionWithBool(bool) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return "anything", nil
	}, nil
}

type multipleArgsArguments struct {
	GetSetterArg GetSetter[any]
	StringArg    string
	FloatArg     float64
	IntArg       int64
}

func functionWithMultipleArgs(GetSetter[interface{}], string, float64, int64) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return "anything", nil
	}, nil
}

type errorFunctionArguments struct{}

func functionThatHasAnError() (ExprFunc[interface{}], error) {
	err := errors.New("testing")
	return func(context.Context, interface{}) (interface{}, error) {
		return "anything", nil
	}, err
}

type enumArguments struct {
	EnumArg Enum
}

func functionWithEnum(Enum) (ExprFunc[interface{}], error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return "anything", nil
	}, nil
}

func createFactory[A any](name string, args A, fn any) Factory[any] {
	createFunction := func(fCtx FunctionContext, oArgs Arguments) (ExprFunc[any], error) {
		fArgs, ok := oArgs.(A)

		if !ok {
			return nil, fmt.Errorf("createFactory args must be of type %T", fArgs)
		}

		funcVal := reflect.ValueOf(fn)

		if funcVal.Kind() != reflect.Func {
			return nil, fmt.Errorf("a non-function value was passed to createFactory")
		}

		argsVal := reflect.ValueOf(fArgs).Elem()
		fnArgs := make([]reflect.Value, argsVal.NumField())

		for i := 0; i < argsVal.NumField(); i++ {
			fnArgs[i] = argsVal.Field(i)
		}

		out := funcVal.Call(fnArgs)

		if !out[1].IsNil() {
			return out[0].Interface().(ExprFunc[any]), out[1].Interface().(error)
		}

		return out[0].Interface().(ExprFunc[any]), nil
	}

	return NewFactory(name, args, createFunction)
}

func defaultFunctionsForTests() map[string]Factory[any] {
	return CreateFactoryMap(
		NewFactory(
			"testing_noop",
			nil,
			func(FunctionContext, Arguments) (ExprFunc[any], error) {
				return functionWithNoArguments()
			},
		),
		createFactory(
			"testing_string_slice",
			&stringSliceArguments{},
			functionWithStringSlice,
		),
		createFactory(
			"testing_float_slice",
			&floatSliceArguments{},
			functionWithFloatSlice,
		),
		createFactory(
			"testing_int_slice",
			&intSliceArguments{},
			functionWithIntSlice,
		),
		createFactory(
			"testing_byte_slice",
			&byteSliceArguments{},
			functionWithByteSlice,
		),
		createFactory[any](
			"testing_getter_slice",
			&getterSliceArguments{},
			functionWithGetterSlice,
		),
		createFactory[any](
			"testing_stringgetter_slice",
			&stringGetterSliceArguments{},
			functionWithStringGetterSlice,
		),
		createFactory[any](
			"testing_durationgetter_slice",
			&durationGetterSliceArguments{},
			functionWithDurationGetterSlice,
		),
		createFactory[any](
			"testing_timegetter_slice",
			&timeGetterSliceArguments{},
			functionWithTimeGetterSlice,
		),
		createFactory[any](
			"testing_stringlikegetter_slice",
			&stringLikeGetterSliceArguments{},
			functionWithStringLikeGetterSlice,
		),
		createFactory[any](
			"testing_floatgetter_slice",
			&floatGetterSliceArguments{},
			functionWithFloatGetterSlice,
		),
		createFactory[any](
			"testing_floatlikegetter_slice",
			&floatLikeGetterSliceArguments{},
			functionWithFloatLikeGetterSlice,
		),
		createFactory[any](
			"testing_intgetter_slice",
			&intGetterSliceArguments{},
			functionWithIntGetterSlice,
		),
		createFactory[any](
			"testing_intlikegetter_slice",
			&intLikeGetterSliceArguments{},
			functionWithIntLikeGetterSlice,
		),
		createFactory[any](
			"testing_pmapgetter_slice",
			&pMapGetterSliceArguments{},
			functionWithPMapGetterSlice,
		),
		createFactory[any](
			"testing_setter",
			&setterArguments{},
			functionWithSetter,
		),
		createFactory[any](
			"testing_getsetter",
			&getSetterArguments{},
			functionWithGetSetter,
		),
		createFactory[any](
			"testing_getter",
			&getterArguments{},
			functionWithGetter,
		),
		createFactory[any](
			"testing_durationgetter",
			&durationGetterArguments{},
			functionWithDurationGetter,
		),
		createFactory[any](
			"testing_timegetter",
			&timeGetterArguments{},
			functionWithTimeGetter,
		),
		createFactory[any](
			"testing_stringgetter",
			&stringGetterArguments{},
			functionWithStringGetter,
		),
		createFactory[any](
			"testing_functiongetter",
			&functionGetterArguments{},
			functionWithFunctionGetter,
		),
		createFactory[any](
			"SHA256",
			&stringGetterArguments{},
			functionWithStringGetter,
		),
		createFactory[any](
			"Sha256",
			&stringGetterArguments{},
			functionWithStringGetter,
		),
		createFactory[any](
			"testing_stringlikegetter",
			&stringLikeGetterArguments{},
			functionWithStringLikeGetter,
		),
		createFactory[any](
			"testing_floatgetter",
			&floatGetterArguments{},
			functionWithFloatGetter,
		),
		createFactory[any](
			"testing_floatlikegetter",
			&floatLikeGetterArguments{},
			functionWithFloatLikeGetter,
		),
		createFactory[any](
			"testing_intgetter",
			&intGetterArguments{},
			functionWithIntGetter,
		),
		createFactory[any](
			"testing_intlikegetter",
			&intLikeGetterArguments{},
			functionWithIntLikeGetter,
		),
		createFactory[any](
			"testing_pmapgetter",
			&pMapGetterArguments{},
			functionWithPMapGetter,
		),
		createFactory[any](
			"testing_string",
			&stringArguments{},
			functionWithString,
		),
		createFactory[any](
			"testing_float",
			&floatArguments{},
			functionWithFloat,
		),
		createFactory[any](
			"testing_int",
			&intArguments{},
			functionWithInt,
		),
		createFactory[any](
			"testing_bool",
			&boolArguments{},
			functionWithBool,
		),
		createFactory[any](
			"testing_multiple_args",
			&multipleArgsArguments{},
			functionWithMultipleArgs,
		),
		createFactory[any](
			"testing_enum",
			&enumArguments{},
			functionWithEnum,
		),
	)
}
