package compiler

import (
	"fmt"
	py "github.com/mbergin/gotopython/pythonast"
	"go/ast"
	"go/token"
)

var (
	pyTrue        = &py.NameConstant{Value: py.True}
	pyFalse       = &py.NameConstant{Value: py.False}
	pyNone        = &py.NameConstant{Value: py.None}
	pyEmptyString = &py.Str{S: `""`}
)

func compileIdent(ident *ast.Ident) py.Expr {
	switch ident.Name {
	case "true":
		return pyTrue
	case "false":
		return pyFalse
	case "nil":
		return pyNone
	default:
		return &py.Name{Id: py.Identifier(ident.Name)}
	}
}

func comparator(t token.Token) (py.CmpOp, bool) {
	switch t {
	case token.EQL:
		return py.Eq, true
	case token.LSS:
		return py.Lt, true
	case token.GTR:
		return py.Gt, true
	case token.NEQ:
		return py.NotEq, true
	case token.LEQ:
		return py.LtE, true
	case token.GEQ:
		return py.GtE, true
	}
	return py.CmpOp(0), false
}

func binOp(t token.Token) (py.Operator, bool) {
	switch t {
	case token.ADD:
		return py.Add, true
	case token.SUB:
		return py.Sub, true
	case token.MUL:
		return py.Mult, true
	case token.QUO:
		return py.FloorDiv, true
	case token.REM:
		return py.Mod, true
	case token.AND:
		return py.BitAnd, true
	case token.OR:
		return py.BitOr, true
	case token.XOR:
		return py.BitXor, true
	case token.SHL:
		return py.LShift, true
	case token.SHR:
		return py.RShift, true
		//case token.AND_NOT: // no &^ in python so special-cased
	}
	return py.Operator(0), false
}

func boolOp(t token.Token) (py.BoolOp, bool) {
	switch t {
	case token.LAND:
		return py.And, true
	case token.LOR:
		return py.Or, true
	}
	return py.BoolOp(0), false
}

func compileBinaryExpr(expr *ast.BinaryExpr) py.Expr {
	if pyCmp, ok := comparator(expr.Op); ok {
		return &py.Compare{
			Left:        compileExpr(expr.X),
			Ops:         []py.CmpOp{pyCmp},
			Comparators: []py.Expr{compileExpr(expr.Y)}}
	}
	if pyOp, ok := binOp(expr.Op); ok {
		return &py.BinOp{Left: compileExpr(expr.X),
			Right: compileExpr(expr.Y),
			Op:    pyOp}
	}
	if pyBoolOp, ok := boolOp(expr.Op); ok {
		return &py.BoolOpExpr{
			Values: []py.Expr{compileExpr(expr.X), compileExpr(expr.Y)},
			Op:     pyBoolOp}
	}
	if expr.Op == token.AND_NOT {
		return &py.BinOp{Left: compileExpr(expr.X),
			Right: &py.UnaryOpExpr{Op: py.Invert, Operand: compileExpr(expr.Y)},
			Op:    py.BitAnd}
	}
	panic(fmt.Sprintf("unknown BinaryExpr Op: %v", expr.Op))
}

func compileBasicLit(expr *ast.BasicLit) py.Expr {
	switch expr.Kind {
	case token.INT, token.FLOAT:
		return &py.Num{N: expr.Value}
	case token.CHAR:
		return &py.Str{S: expr.Value}
	case token.STRING:
		return &py.Str{S: expr.Value}
	}
	panic(fmt.Sprintf("unknown BasicLit kind: %v", expr.Kind))
}

func compileUnaryExpr(expr *ast.UnaryExpr) py.Expr {
	switch expr.Op {
	case token.NOT:
		return &py.UnaryOpExpr{Op: py.Not, Operand: compileExpr(expr.X)}
	case token.AND: // address of
		return compileExpr(expr.X)
	case token.ADD:
		return &py.UnaryOpExpr{Op: py.UAdd, Operand: compileExpr(expr.X)}
	case token.SUB:
		return &py.UnaryOpExpr{Op: py.USub, Operand: compileExpr(expr.X)}
	case token.XOR:
		return &py.UnaryOpExpr{Op: py.Invert, Operand: compileExpr(expr.X)}
	}
	panic(fmt.Sprintf("unknown UnaryExpr: %v", expr.Op))
}

func compileCompositeLit(expr *ast.CompositeLit) py.Expr {
	switch typ := expr.Type.(type) {
	case *ast.Ident:
		var args []py.Expr
		var keywords []py.Keyword
		if len(expr.Elts) > 0 {
			if _, ok := expr.Elts[0].(*ast.KeyValueExpr); ok {
				for _, elt := range expr.Elts {
					kv := elt.(*ast.KeyValueExpr)
					id := identifier(kv.Key.(*ast.Ident))
					keyword := py.Keyword{
						Arg:   &id,
						Value: compileExpr(kv.Value)}
					keywords = append(keywords, keyword)
				}
			} else {
				args = make([]py.Expr, len(expr.Elts))
				for i, elt := range expr.Elts {
					args[i] = compileExpr(elt)
				}
			}
		}
		return &py.Call{
			Func:     compileIdent(typ),
			Args:     args,
			Keywords: keywords,
		}
	case *ast.ArrayType:
		elts := make([]py.Expr, len(expr.Elts))
		for i, elt := range expr.Elts {
			elts[i] = compileExpr(elt)
		}
		return &py.List{Elts: elts}
	case *ast.MapType:
		keys := make([]py.Expr, len(expr.Elts))
		values := make([]py.Expr, len(expr.Elts))
		for i, elt := range expr.Elts {
			kv := elt.(*ast.KeyValueExpr)
			keys[i] = compileExpr(kv.Key)
			values[i] = compileExpr(kv.Value)
		}
		return &py.Dict{Keys: keys, Values: values}
	default:
		panic(fmt.Sprintf("Unknown composite literal type: %T", expr.Type))
	}
}

func compileSelectorExpr(expr *ast.SelectorExpr) py.Expr {
	return &py.Attribute{
		Value: compileExpr(expr.X),
		Attr:  identifier(expr.Sel),
	}
}

func compileCallExpr(expr *ast.CallExpr) py.Expr {
	switch fun := expr.Fun.(type) {
	case *ast.Ident:
		switch fun.Name {
		// TODO need to use proper name resolution to make sure these
		// are really calls to builtin functions and not user-defined
		// functions that hide them.
		case "make":
			typ := expr.Args[0]
			switch t := typ.(type) {
			case *ast.ArrayType:
				length := expr.Args[1]
				// This is a list comprehension rather than [<nil value>] * length
				// because in the case when T is not a primitive type,
				// every element in the list needs to be a different object.
				return &py.ListComp{
					Elt: nilValue(t.Elt),
					Generators: []py.Comprehension{
						py.Comprehension{
							Target: &py.Name{Id: py.Identifier("_")},
							Iter: &py.Call{
								Func: pyRange,
								Args: []py.Expr{compileExpr(length)},
							},
						},
					},
				}
			case *ast.MapType:
				return &py.Dict{}
			default:
				panic("bad type in make()")
			}
		}
	case *ast.ArrayType, *ast.ChanType, *ast.FuncType,
		*ast.InterfaceType, *ast.MapType, *ast.StructType:
		// TODO implement type conversions
		return compileExpr(expr.Args[0])
	}
	return &py.Call{
		Func: compileExpr(expr.Fun),
		Args: compileExprs(expr.Args),
	}
}
func compileSliceExpr(slice *ast.SliceExpr) py.Expr {
	return &py.Subscript{
		Value: compileExpr(slice.X),
		Slice: &py.RangeSlice{
			Lower: compileExpr(slice.Low),
			Upper: compileExpr(slice.High),
		}}
}

func compileIndexExpr(expr *ast.IndexExpr) py.Expr {
	return &py.Subscript{
		Value: compileExpr(expr.X),
		Slice: &py.Index{Value: compileExpr(expr.Index)},
	}
}

func compileExpr(expr ast.Expr) py.Expr {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *ast.UnaryExpr:
		return compileUnaryExpr(e)
	case *ast.BinaryExpr:
		return compileBinaryExpr(e)
	case *ast.Ident:
		return compileIdent(e)
	case *ast.BasicLit:
		return compileBasicLit(e)
	case *ast.ParenExpr:
		return compileExpr(e.X)
	case *ast.CompositeLit:
		return compileCompositeLit(e)
	case *ast.SelectorExpr:
		return compileSelectorExpr(e)
	case *ast.CallExpr:
		return compileCallExpr(e)
	case *ast.IndexExpr:
		return compileIndexExpr(e)
	case *ast.SliceExpr:
		return compileSliceExpr(e)
	}
	panic(fmt.Sprintf("unknown Expr: %T", expr))
}

func compileExprs(exprs []ast.Expr) []py.Expr {
	var pyExprs []py.Expr
	for _, result := range exprs {
		pyExprs = append(pyExprs, compileExpr(result))
	}
	return pyExprs
}

func makeTuple(pyExprs []py.Expr) py.Expr {
	switch len(pyExprs) {
	case 0:
		return nil
	case 1:
		return pyExprs[0]
	default:
		return &py.Tuple{Elts: pyExprs}
	}
}

func compileExprsTuple(exprs []ast.Expr) py.Expr {
	return makeTuple(compileExprs(exprs))
}
