package router

import (
	"sync"
)

type Router struct {
	id       string
	ns       *Namespace
	sessions map[string]Session
	lock     sync.RWMutex
}

func NewRouter(ns *Namespace, id string) *Router {
	return &Router{
		ns:       ns,
		id:       id,
		sessions: make(map[string]Session),
	}
}

func (r *Router) ID() string {
	return r.id
}

func (r *Router) AddSessionIfNotExists(session Session) bool {
	r.lock.Lock()
	defer r.lock.Unlock()
	if _, ok := r.sessions[session.ID()]; !ok {
		r.sessions[session.ID()] = session
		return true
	}

	return false
}

func (r *Router) DeleteSession(session Session) {
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.sessions, session.ID())
}

func (r *Router) Namespace() *Namespace {
	return r.ns
}
