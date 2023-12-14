package rtsp

import (
	"errors"
	"strconv"
	"strings"
)

type Url struct {
	Scheme   string
	Host     string
	Port     int
	Path     string
	User     string
	Password string
	Args     map[string]string
}

func (u *Url) Parse(s string) error {
	u.Scheme = ""
	u.Host = ""
	u.Port = 0
	u.Path = ""
	u.User = ""
	u.Password = ""
	u.Args = make(map[string]string)

	// scheme
	i := strings.Index(s, "://")
	if i < 0 {
		return errors.New("invalid url")
	}
	u.Scheme = s[:i]
	s = s[i+3:]

	// user:password
	i = strings.Index(s, "@")
	if i >= 0 {
		up := s[:i]
		s = s[i+1:]

		i = strings.Index(up, ":")
		if i >= 0 {
			u.User = up[:i]
			u.Password = up[i+1:]
		} else {
			u.User = up
		}
	}

	// host:port
	i = strings.Index(s, "/")
	if i < 0 {
		return errors.New("invalid url")
	}
	hostport := s[:i]
	s = s[i:]

	i = strings.Index(hostport, ":")
	if i >= 0 {
		u.Host = hostport[:i]
		u.Port, _ = strconv.Atoi(hostport[i+1:])
	} else {
		u.Host = hostport
		u.Port = 554
	}

	// path
	i = strings.Index(s, "?")
	if i >= 0 {
		u.Path = s[:i]
		s = s[i+1:]

		// args
		for _, arg := range strings.Split(s, "&") {
			i = strings.Index(arg, "=")
			if i >= 0 {
				u.Args[arg[:i]] = arg[i+1:]
			} else {
				u.Args[arg] = ""
			}
		}
	} else {
		u.Path = s
	}

	return nil
}

func (u *Url) String() string {
	s := u.Scheme + "://"

	if u.User != "" {
		s += u.User
		if u.Password != "" {
			s += ":" + u.Password
		}
		s += "@"
	}

	s += u.Host
	if u.Port != 0 {
		s += ":" + strconv.Itoa(u.Port)
	}

	s += u.Path

	if len(u.Args) > 0 {
		s += "?"
		for k, v := range u.Args {
			s += k
			if v != "" {
				s += "=" + v
			}
			s += "&"
		}
		s = s[:len(s)-1]
	}

	return s
}

func (u *Url) SetArg(key, value string) {
	u.Args[key] = value
}

func (u *Url) GetArg(key string) string {
	return u.Args[key]
}

func (u *Url) DelArg(key string) {
	delete(u.Args, key)
}

func (u *Url) HasArg(key string) bool {
	_, ok := u.Args[key]
	return ok
}

func (u *Url) SetArgs(args map[string]string) {
	u.Args = args
}

func (u *Url) GetArgs() map[string]string {
	return u.Args
}

func (u *Url) ClearArgs() {
	u.Args = make(map[string]string)
}

func (u *Url) SetPath(path string) {
	u.Path = path
}

func (u *Url) GetPath() string {
	return u.Path
}

func (u *Url) SetHost(host string) {
	u.Host = host
}

func (u *Url) GetHost() string {
	return u.Host
}

func (u *Url) SetPort(port int) {
	u.Port = port
}

func (u *Url) GetPort() int {
	return u.Port
}

func (u *Url) SetUser(user string) {
	u.User = user
}

func (u *Url) GetUser() string {
	return u.User
}

func (u *Url) SetPassword(password string) {
	u.Password = password
}

func (u *Url) GetPassword() string {
	return u.Password
}
