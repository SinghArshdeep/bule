package lib

import (
	"fmt"
	"strconv"
	"strings"
)

type Program struct {
	Rules             []Rule
	Constants         map[string]int
	PredicateToTuples map[Predicate][][]int
	PredicateToArity  map[Predicate]int
	PredicateFact     map[string]bool
	GroundFacts       map[Predicate]bool
	Search            map[Predicate]bool
	Alternation       [][]Literal
	existQ            map[int][]Literal
	forallQ           map[int][]Literal
}

type Rule struct {
	initialTokens  []Token
	LineNumber     int
	Parent         *Rule
	Generator      []Literal
	Constraints    []Constraint
	Literals       []Literal
	Iterators      []Iterator
	IsQuestionMark bool // if final token is tokenQuestionMark then it generates, otherwise tokenDot
}

// is a math expression that evaluates to true or false
// Constraints can contain variables
// supported are <,>,<=,>=,==
// E.g..: A*3v<=v5-2*R/7#mod3.
type Constraint struct {
	LeftTerm   Term
	Comparison tokenKind
	RightTerm  Term
}

type Clause []Literal

type Iterator struct {
	Constraints []Constraint
	Literals    []Literal
	Head        Literal
}

type Literal struct {
	Neg   bool
	Fact  bool // if true then search variable with parenthesis () otherwise a fact with brackets []
	Name  Predicate
	Terms []Term
}

func (literal *Literal) IsGround() bool {
	return literal.FreeVars().IsEmpty()
}

type Term string

type Predicate string

func (r *Rule) IsGround() bool {
	return r.FreeVars().IsEmpty()
}

func (r *Rule) IsFact() bool {
	return len(r.Literals) == 1 && r.Literals[0].Fact
}

func IsMarkedAsFree(v string) bool {
	return strings.HasPrefix(v, "_")
}

func (constraint Constraint) Copy() (cons Constraint) {
	cons = constraint
	cons.LeftTerm = constraint.LeftTerm
	cons.RightTerm = constraint.RightTerm
	return cons
}

func (literal Literal) Copy() Literal {
	t := make([]Term, len(literal.Terms))
	copy(t, literal.Terms)
	literal.Terms = t
	return literal
}

// Deep Copy
func (gen Iterator) Copy() (newGen Iterator) {
	newGen = gen
	newGen.Head = gen.Head.Copy()
	newGen.Constraints = []Constraint{}
	newGen.Literals = []Literal{}
	for _, c := range gen.Constraints {
		newGen.Constraints = append(newGen.Constraints, c.Copy())
	}
	for _, l := range gen.Literals {
		newGen.Literals = append(newGen.Literals, l.Copy())
	}
	return
}

// Deep Copy
func (rule Rule) Copy() (newRule Rule) {
	newRule = rule
	newRule.Constraints = []Constraint{}
	newRule.Literals = []Literal{}
	newRule.Iterators = []Iterator{}
	for _, c := range rule.Constraints {
		newRule.Constraints = append(newRule.Constraints, c.Copy())
	}
	for _, l := range rule.Literals {
		newRule.Literals = append(newRule.Literals, l.Copy())
	}
	for _, g := range rule.Iterators {
		newRule.Iterators = append(newRule.Iterators, g.Copy())
	}
	return
}

func (constraint *Constraint) String() string {
	return string(constraint.LeftTerm) + ComparisonString(constraint.Comparison) + string(constraint.RightTerm)
}

func (name Predicate) String() string {
	return string(name)
}

func (term Term) String() string {
	return string(term)
}

func (g Iterator) String() string {
	sb := strings.Builder{}
	sb.WriteString(g.Head.String())
	sb.WriteString(":")
	for _, c := range g.Constraints {
		sb.WriteString(c.String())
		sb.WriteString(":")
	}
	for _, l := range g.Literals {
		sb.WriteString(l.String())
		sb.WriteString(":")
	}
	return strings.TrimSuffix(sb.String(), ":")
}

func (literal Literal) String() string {
	var s string
	var opening string
	var closing string
	if literal.Fact {
		opening = "("
		closing = ")"
	} else {
		opening = "["
		closing = "]"
	}

	if literal.Neg == true {
		s = "~"
	}
	s = s + literal.Name.String() + opening
	for i, x := range literal.Terms {
		s += x.String()
		if i < len(literal.Terms)-1 {
			s += ","
		}
	}
	return s + closing
}

func (r *Rule) Debug() string {
	return r.String() + "  %% #" + strconv.Itoa(r.LineNumber)
}

func (r *Rule) String() string {

	sb := strings.Builder{}

	for _, l := range r.Generator {
		sb.WriteString(l.String())
		sb.WriteString(", ")
	}

	for _, c := range r.Constraints {
		sb.WriteString(c.String())
		sb.WriteString(", ")
	}
	tmp := strings.TrimSuffix(sb.String(), ", ")
	sb.Reset()
	sb.WriteString(tmp)

	sb.WriteString(" => ")

	for _, g := range r.Iterators {
		sb.WriteString(g.String())
		sb.WriteString(", ")
	}

	for _, l := range r.Literals {
		sb.WriteString(l.String())
		sb.WriteString(", ")
	}
	tmp = strings.TrimSuffix(sb.String(), ", ")
	sb.Reset()
	sb.WriteString(tmp)
	if r.IsQuestionMark {
		sb.WriteString("?")
	} else {
		sb.WriteString(".")
	}
	return sb.String()
}

func (p *Program) Debug() {
	fmt.Println("constants:", p.Constants)
	fmt.Println("groundFacts", p.GroundFacts)
	fmt.Println("PredicatFact", p.PredicateFact)
	fmt.Println("PredicatsToTuples", p.PredicateToTuples)
	for _, r := range p.Rules {
		fmt.Println(r.Debug())
	}
}

func (p *Program) PrintDebug(level int) {
	if DebugLevel >= level {
		p.PrintFacts()
		p.PrintRules()
	}
}

func (p *Program) PrintTuples() {

	for pred, tuples := range p.PredicateToTuples {
		fmt.Println(pred.String(), ": ")
		for _, t := range tuples {
			fmt.Println("\t", t)
		}
	}

}

func (p *Program) Print() {
	p.PrintQuantification()
	p.PrintRules()
}

type LiteralError struct {
	L       Literal
	R       Rule
	Message string
}

func (err LiteralError) Error() string {
	var sb strings.Builder
	sb.WriteString(err.Message + ":\n")
	sb.WriteString("Literal " + err.L.String() + "\n")
	sb.WriteString("Rule " + err.R.String() + "\n")
	return sb.String()
}

func (p *Program) CheckNoRemainingFacts() error {
	for _, r := range p.Rules {
		for _, l := range r.Literals {
			if !l.Fact {
				return LiteralError{
					l,
					r,
					fmt.Sprintf("Literals that are used in search should have paranthesis () and not brackets []. \n" +
						"They need to be marked as such!"),
				}
			}
		}
	}
	return nil
}

func (p *Program) CheckFactsInIterators() error {
	for _, r := range p.Rules {
		for _, g := range r.Iterators {
			for _, l := range g.Literals {
				if l.Fact {
					return LiteralError{
						l,
						r,
						fmt.Sprintf("In iterator there is a search literal used as a iterator but has to be fact!\n"),
					}
				}
			}
		}
	}
	return nil
}

func (p *Program) CheckArityOfLiterals() error {
	p.PredicateToArity = make(map[Predicate]int)
	for _, r := range p.Rules {
		for _, l := range r.Literals {
			if n, ok := p.PredicateToArity[l.Name]; ok {
				if n != len(l.Terms) {
					return LiteralError{l, r,
						fmt.Sprintf("Literal with arity %d already occurs in program with arity %d. \n "+
							"Bule predicat to arity has to be unique.", len(l.Terms), n)}
				}
			} else {
				p.PredicateToArity[l.Name] = len(l.Terms)
			}
		}
	}
	return nil
}

func (p *Program) CheckSearch() error {
	p.PredicateToArity = make(map[Predicate]int)
	for _, r := range p.Rules {
		for _, l := range r.Literals {
			if n, ok := p.PredicateToArity[l.Name]; ok {
				if n != len(l.Terms) {
					return LiteralError{l, r,
						fmt.Sprintf("Literal with arity %d already occurs in program with arity %d. \n "+
							"Bule predicat to arity has to be unique.", len(l.Terms), n)}
				}
			} else {
				p.PredicateToArity[l.Name] = len(l.Terms)
			}
		}
	}
	return nil
}

func (p *Program) PrintRules() {
	for _, r := range p.Rules {
		fmt.Print(r.String())
		if DebugLevel > 0 {
			fmt.Print(" % line: ", r.LineNumber)
		}
		fmt.Println()
	}
}

func (p *Program) PrintFacts() {
	for pred := range p.GroundFacts {
		for _, tuple := range p.PredicateToTuples[pred] {
			fmt.Print(pred)
			for i, t := range tuple {
				if i == 0 {
					fmt.Print("[")
				}
				fmt.Print(t)
				if i == len(tuple)-1 {
					fmt.Print("]")
				} else {
					fmt.Print(",")
				}
			}
			fmt.Println(".")
		}
	}
}

func (p *Program) PrintQuantification() {
	for i, quantifier := range p.Alternation {
		if i%2 == 0 {
			fmt.Print("e ")
		} else {
			fmt.Print("a ")
		}
		for _, v := range quantifier {
			fmt.Print(v, " ")
		}
		fmt.Println()
	}
}

// Translates forallQ and existQ into quantification
func (p *Program) MergeConsecutiveQuantificationLevels() {

	maxIndex := -1

	for k := range p.forallQ {
		if maxIndex < k {
			maxIndex = k
		}
	}
	for k := range p.existQ {
		if maxIndex < k {
			maxIndex = k
		}
	}

	last := "e"
	var acc []Literal

	for i := -1; i <= maxIndex; i++ {

		if atoms, ok := p.forallQ[i]; ok {
			if last == "a" {
				acc = append(acc, atoms...)
			} else {
				p.Alternation = append(p.Alternation, acc)
				last = "a"
				acc = atoms
			}
		}
		if atoms, ok := p.existQ[i]; ok {
			if last == "e" {
				acc = append(acc, atoms...)
			} else {
				p.Alternation = append(p.Alternation, acc)
				last = "e"
				acc = atoms
			}
		}
	}
	if len(acc) > 0 {
		p.Alternation = append(p.Alternation, acc)
	}
}
