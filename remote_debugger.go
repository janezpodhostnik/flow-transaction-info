package main

import (
	"github.com/google/pprof/profile"
	"github.com/onflow/cadence"
	"github.com/onflow/flow-go/fvm"
	"github.com/onflow/flow-go/fvm/state"
	"github.com/onflow/flow-go/model/flow"
	"github.com/rs/zerolog"
	"os"
	"path/filepath"
)

type RemoteDebugger struct {
	vm   *fvm.VirtualMachine
	ctx  fvm.Context
	view state.View

	profileBuilder *ProfileBuilder
}

func NewRemoteDebugger(
	view *RemoteView,
	chain flow.Chain,
	directory string,
	logger zerolog.Logger) *RemoteDebugger {
	vm := fvm.NewVirtualMachine()

	profileBuilder := NewProfileBuilder(
		directory,
	)

	// no signature processor here
	// TODO Maybe we add fee-deduction step as well
	ctx := fvm.NewContext(
		fvm.WithLogger(logger),
		fvm.WithChain(chain),
		fvm.WithTransactionProcessors(fvm.NewTransactionInvoker()),
	)

	return &RemoteDebugger{
		ctx:            ctx,
		vm:             vm,
		view:           view,
		profileBuilder: profileBuilder,
	}
}

// RunTransaction runs the transaction given the latest sealed block data
func (d *RemoteDebugger) RunTransaction(txBody *flow.TransactionBody) (txErr, processError error) {
	blockCtx := fvm.NewContextFromParent(d.ctx, fvm.WithBlockHeader(d.ctx.BlockHeader))
	tx := fvm.Transaction(txBody, 0)
	err := d.vm.Run(blockCtx, tx, d.view)
	if err != nil {
		return nil, err
	}
	return tx.Err, nil
}

func (d *RemoteDebugger) RunScript(script *fvm.ScriptProcedure) (value cadence.Value, scriptError, processError error) {
	scriptCtx := fvm.NewContextFromParent(d.ctx, fvm.WithBlockHeader(d.ctx.BlockHeader))
	err := d.vm.Run(scriptCtx, script, d.view)
	if err != nil {
		return nil, nil, err
	}
	return script.Value, script.Err, nil
}

func (d *RemoteDebugger) Close() error {
	return d.profileBuilder.Close()
}

type ProfileBuilder struct {
	Profile            *profile.Profile
	profileFunctionMap map[string]uint64
	lastComputation    uint64
	lastInteraction    uint64
	profileLocationMap map[string]uint64

	nextLocID uint64
	nextFunID uint64
	directory string
}

func NewProfileBuilder(directory string) *ProfileBuilder {
	// https://www.polarsignals.com/blog/posts/2021/08/03/diy-pprof-profiles-using-go/
	p := &profile.Profile{
		Function: []*profile.Function{},
		Location: []*profile.Location{},
	}
	p.SampleType = []*profile.ValueType{
		// {
		// 	Type: "execution effort",
		// 	Unit: "effort",
		// },
		{
			Type: "interaction used",
			Unit: "bytes",
		},
	}

	return &ProfileBuilder{
		Profile:            p,
		profileFunctionMap: make(map[string]uint64),
		profileLocationMap: make(map[string]uint64),
		directory:          directory,
	}
}

func (p *ProfileBuilder) Close() error {
	filename := p.directory + "/profile.pb.gz"
	err := os.MkdirAll(filepath.Dir(filename), os.ModePerm)
	if err != nil {
		return err
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		err := f.Close()
		if err != nil {
			// log
		}
	}()

	// Write the profile to the file.
	err = p.Profile.Write(f)
	if err != nil {
		return err
	}
	return nil
}

//
// func (p *ProfileBuilder) OnCadenceStatement(fvmEnv runtime2.Environment, inter *interpreter.Interpreter, statement ast.Statement) {
// 	stack := inter.CallStack()
// 	if len(stack) == 0 {
// 		// what now?
// 		return
// 	}
//
// 	// newComputation := fvmEnv.(environment.Environment).ComputationUsed()
// 	// computation := newComputation - p.lastComputation
// 	// p.lastComputation = newComputation
//
// 	newInteraction := fvmEnv.(environment.Environment).InteractionUsed()
// 	interaction := newInteraction - p.lastInteraction
// 	p.lastInteraction = newInteraction
//
// 	locationIds := make([]uint64, 0, len(stack))
//
// 	// var lastFrame interpreter.Invocation
// 	for _, frame := range stack {
// 		// lastFrame = frame
// 		fn := p.toFunction(inter, frame)
// 		if fn == nil {
// 			continue
// 		}
// 		fnIndex, ok := p.profileFunctionMap[p.fnID(fn)]
// 		if !ok {
// 			p.Profile.Function = append(p.Profile.Function, fn)
// 			p.Profile.Location = append(p.Profile.Location,
// 				&profile.Location{
// 					ID:      p.nextLocID + 1,
// 					Address: p.nextLocID + 1,
// 					Line: []profile.Line{
// 						{
// 							Function: fn,
// 							Line:     fn.StartLine,
// 						},
// 					},
// 				},
// 			)
// 			fnIndex = p.nextFunID
// 			p.profileFunctionMap[p.fnID(fn)] = p.nextFunID
// 			p.profileLocationMap[p.fnID(fn)] = p.nextLocID
// 			p.nextFunID++
// 			p.nextLocID++
// 		}
// 		locationIds = append(locationIds, fnIndex)
// 	}
//
// 	locations := make([]*profile.Location, 0, len(locationIds))
// 	// revers iterate locations
// 	for i := len(locationIds) - 1; i >= 0; i-- {
// 		locations = append(locations, p.Profile.Location[locationIds[i]])
// 	}
//
// 	p.Profile.Sample = append(p.Profile.Sample, &profile.Sample{
// 		Location: locations,
// 		Value: []int64{
// 			// int64(computation),
// 			int64(interaction),
// 		},
// 	})
// }
//
// func (p *ProfileBuilder) fnID(fn *profile.Function) string {
// 	return fn.Filename + "_" + fn.Name
// }
//
// func (p *ProfileBuilder) toFunction(inter *interpreter.Interpreter, frame interpreter.Invocation) *profile.Function {
// 	filename := frame.Self.StaticType(inter).String()
// 	name := ""
// 	line := int64(0)
//
// 	if frame.LocationRange.HasPosition != nil {
// 		switch frame.LocationRange.HasPosition.(type) {
// 		case *ast.InvocationExpression:
// 			expression := frame.LocationRange.HasPosition.(*ast.InvocationExpression)
// 			line = int64(expression.InvokedExpression.StartPosition().Line)
//
// 			switch expression.InvokedExpression.(type) {
// 			case *ast.MemberExpression:
// 				me := expression.InvokedExpression.(*ast.MemberExpression)
// 				name = me.Identifier.String()
// 			case *ast.IdentifierExpression:
// 				ie := expression.InvokedExpression.(*ast.IdentifierExpression)
// 				name = ie.Identifier.String()
// 			default:
// 				panic("")
// 			}
// 		default:
// 			field := reflect.ValueOf(inter).Elem().FieldByName("statement")
//
// 			statement := reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(ast.Statement)
//
// 			switch statement.(type) {
// 			case *ast.VariableDeclaration:
// 				return nil
// 			case *ast.AssignmentStatement:
// 				return nil
// 			case *ast.ReturnStatement:
// 				return nil
// 			case *ast.IfStatement:
// 				return nil
// 			}
//
// 			line = int64(statement.StartPosition().Line)
// 			name = statement.String()
//
// 			println("unknown type")
//
// 		}
// 	}
//
// 	return &profile.Function{
// 		ID:         p.nextFunID + 1,
// 		Name:       name,
// 		SystemName: name,
// 		Filename:   filename,
// 		StartLine:  line,
// 	}
// }
