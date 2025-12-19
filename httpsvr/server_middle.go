package httpsvr

func (s *EasyServer) AddMiddleHead(middle MiddleHandle) {
	s.headMiddles = append(s.headMiddles, middle)
}

func (s *EasyServer) AddMiddleTail(middle MiddleHandle) {
	s.tailMiddles = append(s.tailMiddles, middle)
}

func (s *EasyServer) appendMiddleware(middles ...MiddleHandle) {
	s.middles = append(s.middles, middles...)
}
