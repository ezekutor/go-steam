package server

// Credentials represents the login information necessary to authenticate a
// mirror account on Steam.
type Credentials struct {
	Login    string
	Password string
}

// Server describes a logical group of accounts that should be authenticated
// against Steam as part of the mirror process.
type Server struct {
	Name        string
	Credentials []Credentials
}

// Accounts exposes the credential list for compatibility with existing
// call-sites.
func (s *Server) Accounts() []Credentials {
	if s == nil {
		return nil
	}
	return s.Credentials
}
