package monocrond

import (
	"net/http"

	"encore.app/gen/daemon/v1/daemonv1connect"
)

//encore:service
type Service struct {
	routes http.Handler
}

//encore:api public raw path=/daemon.v1.ExecuteRequest/*endpoint
func (s *Service) ExecuteRequest(w http.ResponseWriter, req *http.Request) {
	s.routes.ServeHTTP(w, req)
}

func initService() (*Service, error) {
	e := &MonocrondServer{}
	mux := http.NewServeMux()
	path, handler := daemonv1connect.NewDaemonServiceHandler(e)
	mux.Handle(path, handler)
	return &Service{routes: mux}, nil
}
