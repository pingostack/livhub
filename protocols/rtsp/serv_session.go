package rtsp

import "sync"

type IServSessionEventListener interface {
	OnDescribe(serv *Serv) error
	OnAnnounce(serv *Serv) error
	OnPause(serv *Serv) error
	OnResume(serv *Serv) error
	OnStream(serv *Serv) error
}

type IServSession interface {
	AddParams(k, v interface{})
	GetParams(k interface{}) (interface{}, bool)
	DeleteParams(k interface{})
	SetEventListener(listener IServSessionEventListener)
	GetEventListener() IServSessionEventListener
	Logger() Logger
}

type ServSession struct {
	params   sync.Map
	listener IServSessionEventListener
}

func NewServSession(listener IServSessionEventListener) *ServSession {
	return &ServSession{
		listener: listener,
	}
}

func (ss *ServSession) AddParams(k, v interface{}) {
	ss.params.Store(k, v)
}

func (ss *ServSession) GetParams(k interface{}) (interface{}, bool) {
	return ss.params.Load(k)
}

func (ss *ServSession) DeleteParams(k interface{}) {
	ss.params.Delete(k)
}

func (ss *ServSession) SetEventListener(listener IServSessionEventListener) {
	ss.listener = listener
}

func (ss *ServSession) GetEventListener() IServSessionEventListener {
	return ss.listener
}

func (ss *ServSession) Logger() Logger {
	return nil
}
