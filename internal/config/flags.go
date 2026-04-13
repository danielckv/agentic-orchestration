package config

type Role string

const (
	RoleResearcher Role = "researcher"
	RoleCoder      Role = "coder"
	RoleReviewer   Role = "reviewer"
	RolePlanner    Role = "planner"
)

var RoleCapabilities = map[Role][]string{
	RoleResearcher: {"web_search", "rag_retrieval", "citation"},
	RoleCoder:      {"file_io", "code_exec", "git"},
	RoleReviewer:   {"reflection", "diff_audit", "consensus"},
	RolePlanner:    {"cot", "tot", "dag_modify"},
}

func ValidRole(s string) bool {
	_, ok := RoleCapabilities[Role(s)]
	return ok
}
