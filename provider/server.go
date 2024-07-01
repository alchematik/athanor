package provider

type Server struct {
	Impl Provider
}

func (s *Server) Get(req GetResourceRequest, res *GetResourceResponse) error {
	r, err := s.Impl.Get(req)
	if err != nil {
		return err
	}

	*res = r
	return nil
}
