package core

import (
	"errors"
	"sync"
)

type serv struct {
	namespaces sync.Map
}

var servInstance *serv

func Serv() *serv {
	return servInstance
}

func init() {
	servInstance = &serv{
		namespaces: sync.Map{},
	}
}

func (s *serv) InitStaticNamespaces(namespaces []NamespaceInfo) {
	for _, ns := range namespaces {
		s.namespaces.Store(ns.Name, NewNamespace(ns.Name, ns.Domain...))
	}
}

func (s *serv) getOrNewNamespaceByDomain(req *PeerReq) (*Namespace, bool) {
	var ns *Namespace
	var newOne bool
	s.namespaces.Range(func(key, value interface{}) bool {
		ns = value.(*Namespace)
		if ns.HasDomain(req.Domain) {
			return false
		}
		return true
	})

	if ns == nil {
		ns = NewNamespace(req.Domain)
		s.namespaces.Store(req.Domain, ns)
		newOne = true
	}

	return ns, newOne
}

func (s *serv) join(req *PeerReq) (*Namespace, *Stream, error) {
	ns, _ := s.getOrNewNamespaceByDomain(req)
	stream, _ := ns.GetOrNewStream(req.StreamID)
	ok := stream.AddSessionIfNotExists(req.Session)
	if !ok {
		return nil, nil, errors.New("session already exists")
	}

	return ns, stream, nil
}

func (s *serv) Publish(req *PublishReq) error {
	ns, stream, err := s.join(&req.PeerReq)
	if err != nil {
		return err
	}

	return nil
}

func (s *serv) Play(req *PlayReq) error {
	ns, stream, err := s.join(&req.PeerReq)
	if err != nil {
		return err
	}

	return nil
}
