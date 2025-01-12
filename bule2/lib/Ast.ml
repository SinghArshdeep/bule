open Printf

include Types.AST
open T

module Print =
struct
  let id x = x
  let cname = id

  let eoperator_aux = function
    | Add -> "+"
    | Div -> "/"
    | Mult -> "*"
    | Max -> "\\max"
    | Min -> "\\min"
    | Log -> "//"
    | Mod -> "#mod"
    | Pow -> "**"
    | Sub -> "-"
  let comparison_operator = function
    | Lt -> "<"
    | Gt -> ">"
    | Leq -> "<="
    | Geq -> ">="
    | Eq -> "=="
    | Neq -> "!="
  let eoperator o = "\\" ^ eoperator_aux o

  let list_tuple f = Print.list' "(" ", " ")" f

  let rec expr = function
    | BinE (e1, bo, e2) -> sprintf "%s %s %s" (inner_expr e1) (eoperator bo) (inner_expr e2)
    | VarE _ | Int _ as e -> inner_expr e
  and inner_expr = function
    | VarE n -> n
    | Int i -> Print.int i
    | BinE _ as e -> sprintf "(%s)" (expr e)
  let rec term : term -> string = function
    | Exp e -> expr e
    | Fun (c, ts) -> sprintf "%s%s" c (list_tuple term ts)
  let atom (name, terms) = sprintf "%s[%s]" name (Print.list' "" ", " "" term terms)
  let rec tuple = function
    | ExpTu e -> expr e
    | FunTu (c, ts) -> sprintf "%s%s" c (list_tuple tuple ts)
    | Range (e1, e2) -> sprintf "%s..%s" (expr e1) (expr e2)
  let atomd (name, tuples) = sprintf "%s[%s]" name (Print.list' "" ", " "" tuple tuples)
  let search_var (name, terms) = sprintf "%s(%s)" name (Print.list' "" ", " "" term terms)
  let searchd (name, tuples) = sprintf "%s(%s)" name (Print.list' "" ", " "" tuple tuples)

  let ground_literal = function
    | In ga -> atom ga
    | Notin ga -> sprintf "~%s" (atom ga)
    | Comparison (t1, c, t2) -> sprintf "%s %s %s" (term t1) (comparison_operator c) (term t2)
    | Set (v, t) -> sprintf "%s := %s" v (term t)

  let glits gls = Print.list' "" ", " "" ground_literal gls
  let literal (pol, var) =
    let pol = if pol then "" else "~" in
    let var = search_var var in
    pol ^ var

  let literals (gls, pol, var) =
    let gls = match gls with
    | [] -> ""
    | _ :: _ -> glits gls ^ " : " in
    gls ^ literal (pol, var)

  let ground_decl (gls, ats) = sprintf "%s :: %s." (glits gls) (Print.list' "" ", " "" atomd ats)
  let search_decl (gls, (exi, depth, vars)) =
    let quant = if exi then "exists" else "forall" in
    let gls = match gls with [] -> "" | _ :: _ -> ", " ^ glits gls in
    let svs = Print.list' "" ", " "" searchd vars in
    sprintf "#%s[%s]%s :: %s?" quant (expr depth) gls svs
  let clause_decl (gls, (hyps, ccls)) = sprintf "%s :: %s -> %s." (glits gls) (Print.list' "" " & " "" literals hyps) (Print.list' "" " | " "" literals ccls)
  let hide_decl (gls, (hide, lits)) =
    let gls = match gls with [] -> "" | _ :: _ -> ", " ^ glits gls in
    let h = if hide then "hide" else "show" in
    sprintf "%%#%s %s :: %s." h gls (Print.list' "" ", " "" literal lits)

  let file { ground; prefix; matrix; hide } =
    sprintf "%s\n%s\n%s\n%s\n"
      (Print.unlines ground_decl ground)
      (Print.unlines search_decl prefix)
      (Print.unlines clause_decl matrix)
      (Print.unlines hide_decl hide)
end

module PA = Types.PARSE.T

let unroll_comparison_chain t l =
  let aux (accu_l, accu_t) (op, t) =
    (Comparison (accu_t, op, t) :: accu_l, t) in
  fst (List.fold_left aux ([], t) l)

let ground_literal = function
  | PA.In a -> [In a]
  | PA.Notin a -> [Notin a]
  | PA.Chain (_, []) -> assert false
  | PA.Chain (t, l) -> unroll_comparison_chain t l
  | PA.Set s -> [Set s]
let literals (gs, b, a) = (List.concat_map ground_literal gs, b, a)
let clause_decl (hyps, ccls) = (List.map literals hyps, List.map literals ccls)
let file (decls : PA.file) =
  let aux (gs, ss, cs, hs) (glits, decl) =
    let glits = List.concat_map ground_literal glits in
    match decl with
    | PA.G gd -> ((glits, gd) :: gs, ss, cs, hs)
    | S sd -> (gs, (glits, sd) :: ss, cs, hs)
    | C cd -> (gs, ss, (glits, clause_decl cd) :: cs, hs)
    | H hd -> (gs, ss, cs, (glits, hd) :: hs) in
  let gs, ss, cs, hs = List.fold_left aux ([], [], [], []) decls in
  let ground, prefix, matrix, hide = List.rev gs, List.rev ss, List.rev cs, List.rev hs in
  { ground;
    prefix;
    matrix;
    hide }
