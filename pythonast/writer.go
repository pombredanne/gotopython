package pythonast

import (
	"fmt"
	"io"
)

type Writer struct {
	out         io.Writer
	indentLevel int
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{out: w}
}

func (w *Writer) WriteModule(m *Module) {
	for _, bodyStmt := range m.Body {
		w.writeStmt(bodyStmt)
		w.newline()
	}
}

func (w *Writer) writeStmts(stmts []Stmt) {
	for i, stmt := range stmts {
		if i > 0 {
			w.newline()
		}
		w.writeStmt(stmt)
	}
}

func (w *Writer) writeStmt(stmt Stmt) {
	switch s := stmt.(type) {
	case *FunctionDef:
		w.functionDef(s)
	case *ClassDef:
		w.classDef(s)
	case *While:
		w.while(s)
	case *Assign:
		w.assign(s)
	case *Return:
		w.ret(s)
	case *Pass:
		w.write("pass")
	case *ExprStmt:
		w.WriteExpr(s.Value)
	case *If:
		w.ifStmt(s)
	case *AugAssign:
		w.augAssign(s)
	case *For:
		w.forLoop(s)
	case *Break:
		w.write("break")
	case *Continue:
		w.write("continue")
	case *Delete:
		w.del(s)
	case *Try:
		w.try(s)
	case *Comment:
		w.comment(s)
	case *DocString:
		w.docstring(s)
	default:
		panic(fmt.Sprintf("unknown Stmt: %T", stmt))
	}
}

func (w *Writer) comment(s *Comment) {
	w.write("#")
	w.write(s.Text)
}

func (w *Writer) docstring(s *DocString) {
	w.write(`"""`)
	w.newline()
	for _, line := range s.Lines {
		w.write(line)
		w.newline()
	}
	w.write(`"""`)
}

func (w *Writer) ret(s *Return) {
	if s.Value != nil {
		w.write("return ")
		w.WriteExpr(s.Value)
	} else {
		w.write("return")
	}
}

func (w *Writer) del(s *Delete) {
	w.write("del ")
	for i, target := range s.Targets {
		if i > 0 {
			w.comma()
		}
		w.WriteExpr(target)
	}
}

func (w *Writer) assign(s *Assign) {
	for i, target := range s.Targets {
		if i > 0 {
			w.comma()
		}
		w.WriteExpr(target)
	}
	w.write(" = ")
	w.WriteExpr(s.Value)
}

func (w *Writer) while(s *While) {
	w.write("while ")
	w.WriteExpr(s.Test)
	w.write(":")
	w.indent()
	w.writeStmts(s.Body)
	w.dedent()
}

func (w *Writer) ifStmt(s *If) {
	w.write("if ")
	w.WriteExpr(s.Test)
	w.write(":")
	w.indent()
	w.writeStmts(s.Body)
	w.dedent()
	if s.Orelse != nil {
		w.newline()
		if elif, ok := s.Orelse[0].(*If); ok {
			w.write("el")
			w.writeStmt(elif)
		} else {
			w.write("else:")
			w.indent()
			w.writeStmts(s.Orelse)
			w.dedent()
		}
	}
}

func (w *Writer) forLoop(s *For) {
	w.write("for ")
	w.WriteExpr(s.Target)
	w.write(" in ")
	w.WriteExpr(s.Iter)
	w.write(":")
	w.indent()
	for i, bodyStmt := range s.Body {
		if i > 0 {
			w.newline()
		}
		w.writeStmt(bodyStmt)
	}
	w.dedent()
}

func (w *Writer) try(s *Try) {
	w.write("try:")
	w.indent()
	w.writeStmts(s.Body)
	w.dedent()
	for _, handler := range s.Handlers {
		w.newline()
		w.write("except")
		if handler.Typ != nil {
			w.write(" ")
			w.WriteExpr(handler.Typ)
			if handler.Name != Identifier("") {
				w.write(" as ")
				w.identifier(handler.Name)
			}
		}
		w.write(":")
		w.indent()
		w.writeStmts(handler.Body)
		w.dedent()
	}
	if len(s.Orelse) > 0 {
		w.newline()
		w.write("else:")
		w.indent()
		w.writeStmts(s.Orelse)
		w.dedent()
	}
	if len(s.Finalbody) > 0 {
		w.newline()
		w.write("finally:")
		w.indent()
		w.writeStmts(s.Finalbody)
		w.dedent()
	}
}

func (w *Writer) augAssign(s *AugAssign) {
	w.WriteExpr(s.Target)
	switch s.Op {
	case Add:
		w.write(" += ")
	case Sub:
		w.write(" -= ")
	case Mult:
		w.write(" *= ")
	case MatMult:
		w.write(" @= ")
	case Div:
		w.write(" /= ")
	case Mod:
		w.write(" %= ")
	case Pow:
		w.write(" **= ")
	case LShift:
		w.write(" <<= ")
	case RShift:
		w.write(" >>= ")
	case BitOr:
		w.write(" |= ")
	case BitXor:
		w.write(" ^= ")
	case BitAnd:
		w.write(" &= ")
	case FloorDiv:
		w.write(" //= ")
	}
	w.WriteExpr(s.Value)
}

func (w *Writer) call(e *Call) {
	prec := e.Precedence()
	w.writeExprPrec(e.Func, prec)
	w.beginParen()
	i := 0
	for _, arg := range e.Args {
		if i != 0 {
			w.comma()
		}
		w.writeExprPrec(arg, prec)
		i++
	}
	for _, kw := range e.Keywords {
		if i != 0 {
			w.comma()
		}
		w.identifier(*kw.Arg)
		w.write("=")
		w.writeExprPrec(kw.Value, prec)
		i++
	}
	w.endParen()
}

func (w *Writer) binOp(e *BinOp) {
	prec := e.Precedence()
	if prec == e.Left.Precedence() && e.Op == Pow {
		w.beginParen()
	}
	w.writeExprPrec(e.Left, prec)
	if prec == e.Left.Precedence() && e.Op == Pow {
		w.endParen()
	}
	w.writeOp(e.Op)
	if prec == e.Right.Precedence() && e.Op != Pow {
		w.beginParen()
	}
	w.writeExprPrec(e.Right, prec)
	if prec == e.Right.Precedence() && e.Op != Pow {
		w.endParen()
	}
}

func (w *Writer) tuple(e *Tuple, parentPrec int) {
	prec := e.Precedence()
	paren := prec < parentPrec
	if !paren && len(e.Elts) == 0 {
		w.beginParen()
	}
	for i, elt := range e.Elts {
		if i > 0 {
			w.comma()
		}
		w.writeExprPrec(elt, 1)
	}
	if len(e.Elts) == 1 {
		w.write(",")
	}
	if !paren && len(e.Elts) == 0 {
		w.endParen()
	}
}

func (w *Writer) WriteExpr(expr Expr) {
	w.writeExprPrec(expr, 0)
}

func (w *Writer) writeExprPrec(expr Expr, parentPrec int) {
	if expr == nil {
		panic("nil expr")
	}
	prec := expr.Precedence()
	paren := prec < parentPrec
	if paren {
		w.beginParen()
	}
	switch e := expr.(type) {
	case *BinOp:
		w.binOp(e)
	case *Name:
		w.identifier(e.Id)
	case *Num:
		w.write(e.N)
	case *Str:
		w.write(e.S)
	case *Compare:
		w.writeExprPrec(e.Left, prec)
		for i := range e.Ops {
			w.writeCmpOp(e.Ops[i])
			w.writeExprPrec(e.Comparators[i], prec)
		}
	case *Tuple:
		w.tuple(e, parentPrec)
	case *Call:
		w.call(e)
	case *Attribute:
		w.writeExprPrec(e.Value, prec)
		w.write(".")
		w.identifier(e.Attr)
	case *NameConstant:
		w.nameConstant(e)
	case *List:
		w.list(e)
	case *Dict:
		w.dict(e)
	case *Subscript:
		w.writeExprPrec(e.Value, prec)
		w.write("[")
		w.slice(e.Slice)
		w.write("]")
	case *BoolOpExpr:
		w.boolOpExpr(e)
	case *UnaryOpExpr:
		w.unaryOpExpr(e)
	case *ListComp:
		w.listComp(e)
	case *Starred:
		w.starred(e)
	case *Lambda:
		w.lambda(e)
	default:
		panic(fmt.Sprintf("unknown Expr: %T", expr))
	}
	if paren {
		w.endParen()
	}
}

func (w *Writer) lambda(e *Lambda) {
	w.write("lambda ")
	w.args(e.Args)
	w.write(": ")
	w.writeExprPrec(e.Body, e.Precedence())
}

func (w *Writer) starred(e *Starred) {
	w.write("*")
	w.writeExprPrec(e.Value, e.Precedence())
}

func (w *Writer) listComp(e *ListComp) {
	w.write("[")
	w.WriteExpr(e.Elt)
	for _, g := range e.Generators {
		w.write(" for ")
		w.WriteExpr(g.Target)
		w.write(" in ")
		w.WriteExpr(g.Iter)
		for _, ifExpr := range g.Ifs {
			w.write(" if ")
			w.WriteExpr(ifExpr)
		}
	}
	w.write("]")
}

func (w *Writer) boolOpExpr(e *BoolOpExpr) {
	w.writeExprPrec(e.Values[0], e.Precedence())
	switch e.Op {
	case Or:
		w.write(" or ")
	case And:
		w.write(" and ")
	}
	w.writeExprPrec(e.Values[1], e.Precedence())
}

func (w *Writer) unaryOpExpr(e *UnaryOpExpr) {
	switch e.Op {
	case Invert:
		w.write("~")
	case Not:
		w.write("not ")
	case UAdd:
		w.write("+")
	case USub:
		w.write("-")
	}
	w.writeExprPrec(e.Operand, e.Precedence())
}

func (w *Writer) slice(s Slice) {
	switch s := s.(type) {
	case *Index:
		w.WriteExpr(s.Value)
	case *RangeSlice:
		if s.Lower != nil {
			w.WriteExpr(s.Lower)
		}
		w.write(":")
		if s.Upper != nil {
			w.WriteExpr(s.Upper)
		}
	default:
		panic(fmt.Sprintf("unknown Slice: %T", s))
	}
}

func (w *Writer) list(l *List) {
	w.write("[")
	for i, elt := range l.Elts {
		if i > 0 {
			w.comma()
		}
		w.writeExprPrec(elt, l.Precedence())
	}
	w.write("]")
}

func (w *Writer) dict(d *Dict) {
	w.write("{")
	for i := range d.Keys {
		if i > 0 {
			w.comma()
		}
		w.writeExprPrec(d.Keys[i], d.Precedence())
		w.write(": ")
		w.writeExprPrec(d.Values[i], d.Precedence())
	}
	w.write("}")
}

func (w *Writer) nameConstant(nc *NameConstant) {
	switch nc.Value {
	case None:
		w.write("None")
	case True:
		w.write("True")
	case False:
		w.write("False")
	default:
		panic(fmt.Sprintf("unknown NameConstant %v", nc.Value))
	}
}

func (w *Writer) writeOp(op Operator) {
	switch op {
	case Add:
		w.write(" + ")
	case Sub:
		w.write(" - ")
	case Mult:
		w.write(" * ")
	case MatMult:
		w.write(" @ ")
	case Div:
		w.write(" / ")
	case Mod:
		w.write(" % ")
	case Pow:
		w.write(" ** ")
	case LShift:
		w.write(" << ")
	case RShift:
		w.write(" >> ")
	case BitOr:
		w.write(" | ")
	case BitXor:
		w.write(" ^ ")
	case BitAnd:
		w.write(" & ")
	case FloorDiv:
		w.write(" // ")
	}
}

func (w *Writer) writeCmpOp(op CmpOp) {
	switch op {
	case Eq:
		w.write(" == ")
	case NotEq:
		w.write(" != ")
	case Lt:
		w.write(" < ")
	case LtE:
		w.write(" <= ")
	case Gt:
		w.write(" > ")
	case GtE:
		w.write(" >= ")
	case Is:
		w.write(" is ")
	case IsNot:
		w.write(" is not ")
	case In:
		w.write(" in ")
	case NotIn:
		w.write(" not in ")
	}
}

func (w *Writer) args(args Arguments) {
	defaultOffset := len(args.Args) - len(args.Defaults)
	for i, arg := range args.Args {
		if i > 0 {
			w.comma()
		}
		w.identifier(arg.Arg)
		if i >= defaultOffset {
			w.write("=")
			w.WriteExpr(args.Defaults[i-defaultOffset])
		}
	}
}

func (w *Writer) functionDef(s *FunctionDef) {
	w.newline()
	w.write("def ")
	w.identifier(s.Name)
	w.beginParen()
	w.args(s.Args)
	w.endParen()
	w.write(":")
	w.indent()
	for i, bodyStmt := range s.Body {
		if i > 0 {
			w.newline()
		}
		w.writeStmt(bodyStmt)
	}
	w.dedent()
}

func (w *Writer) classDef(s *ClassDef) {
	w.newline()
	w.write("class ")
	w.identifier(s.Name)
	if len(s.Bases) > 0 {
		w.beginParen()
		for i, base := range s.Bases {
			if i > 0 {
				w.comma()
			}
			w.WriteExpr(base)
		}
		w.endParen()
	}
	w.write(":")
	w.indent()
	for i, bodyStmt := range s.Body {
		if i > 0 {
			w.newline()
		}
		w.writeStmt(bodyStmt)
	}
	w.dedent()
}

func (w *Writer) identifier(i Identifier) {
	w.write(string(i))
}

func (w *Writer) comma() {
	w.write(", ")
}

func (w *Writer) beginParen() {
	w.write("(")
}

func (w *Writer) endParen() {
	w.write(")")
}

func (w *Writer) indent() {
	w.indentLevel++
	w.newline()
}

func (w *Writer) newline() {
	w.write("\n")
	for i := 0; i < w.indentLevel; i++ {
		w.write("    ")
	}
}

func (w *Writer) dedent() {
	w.indentLevel--
}

func (w *Writer) write(s string) {
	w.out.Write([]byte(s))
}
