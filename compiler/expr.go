package compiler

import (
	"fmt"
	py "github.com/mbergin/gotopython/pythonast"
	"go/ast"
	"go/token"
)

func compileIdent(ident *ast.Ident) py.Expr {
	switch ident.Name {
	case "true":
		return &py.NameConstant{Value: py.True}
	case "false":
		return &py.NameConstant{Value: py.False}
	case "nil":
		return &py.NameConstant{Value: py.None}
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

func augmentedOp(t token.Token) py.Operator {
	switch t {
	case token.ADD_ASSIGN: // +=
		return py.Add
	case token.SUB_ASSIGN: // -=
		return py.Sub
	case token.MUL_ASSIGN: // *=
		return py.Mult
	case token.QUO_ASSIGN: // /=
		return py.FloorDiv
	case token.REM_ASSIGN: // %=
		return py.Mod
	case token.AND_ASSIGN: // &=
		return py.BitAnd
	case token.OR_ASSIGN: // |=
		return py.BitOr
	case token.XOR_ASSIGN: // ^=
		return py.BitXor
	case token.SHL_ASSIGN: // <<=
		return py.LShift
	case token.SHR_ASSIGN: // >>=
		return py.RShift
		//case AND_NOT_ASSIGN: // &^=
	default:
		panic(fmt.Sprintf("augmentedOp bad token %v", t))
	}
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
	panic(fmt.Sprintf("unknown BasicLit: %T", expr))
}

func compileUnaryExpr(expr *ast.UnaryExpr) py.Expr {
	switch expr.Op {
	case token.NOT:
		return &py.UnaryOpExpr{Op: py.Not, Operand: compileExpr(expr.X)}
	case token.AND: // address of
		return compileExpr(expr.X)
	case token.SUB:
		return &py.UnaryOpExpr{Op: py.USub, Operand: compileExpr(expr.X)}
	}
	panic(fmt.Sprintf("unknown UnaryExpr: %v", expr.Op))
}

func compileCompositeLit(expr *ast.CompositeLit) py.Expr {
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
			for _, elt := range expr.Elts {
				args = append(args, compileExpr(elt))
			}
		}
	}
	return &py.Call{
		Func:     compileIdent(expr.Type.(*ast.Ident)),
		Args:     args,
		Keywords: keywords,
	}
}

func compileSelectorExpr(expr *ast.SelectorExpr) py.Expr {
	return &py.Attribute{
		Value: compileExpr(expr.X),
		Attr:  identifier(expr.Sel),
	}
}

func compileCallExpr(expr *ast.CallExpr) py.Expr {
	if ident, ok := expr.Fun.(*ast.Ident); ok {
		switch ident.Name {
		case "make":
			typ := expr.Args[0]
			switch t := typ.(type) {
			case *ast.ArrayType:
				length := expr.Args[1]
				return &py.BinOp{
					Left:  &py.List{Elts: []py.Expr{nilValue(t.Elt)}},
					Op:    py.Mult,
					Right: compileExpr(length),
				}
			default:
				panic("bad type in make()")
			}
		}
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

func compileExprsTuple(exprs []ast.Expr) py.Expr {
	pyExprs := compileExprs(exprs)
	if len(pyExprs) == 1 {
		return pyExprs[0]
	}
	return &py.Tuple{Elts: pyExprs}
}