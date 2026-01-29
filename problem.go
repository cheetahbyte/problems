package problem

import (
	"encoding/json"
	"errors"
	"maps"
	"net/http"
)

const MediaType = "application/problem+json"

type Option interface{ apply(*Problem) }

type optionFunc func(*Problem)

func (f optionFunc) apply(p *Problem) { f(p) }

type Problem struct {
	data  map[string]any
	cause error
}

func New(opts ...Option) *Problem {
	p := &Problem{data: make(map[string]any)}
	for _, opt := range opts {
		if opt != nil {
			opt.apply(p)
		}
	}
	return p
}

func Of(status int) *Problem {
	return New(Status(status), Title(http.StatusText(status)))
}

func (p *Problem) Append(opts ...Option) *Problem {
	if p == nil {
		return New(opts...)
	}
	if p.data == nil {
		p.data = make(map[string]any)
	}
	for _, opt := range opts {
		if opt != nil {
			opt.apply(p)
		}
	}
	return p
}

func (p *Problem) Data() map[string]any {
	if p == nil || p.data == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(p.data))
	maps.Copy(out, p.data)
	return out
}

func (p *Problem) Get(key string) (any, bool) {
	if p == nil || p.data == nil {
		return nil, false
	}
	v, ok := p.data[key]
	return v, ok
}

func (p *Problem) JSON() []byte {
	b, _ := json.Marshal(p)
	return b
}

func (p *Problem) JSONString() string { return string(p.JSON()) }

func (p *Problem) Error() string { return p.JSONString() }

func (p *Problem) Unwrap() error {
	if p == nil {
		return nil
	}
	return p.cause
}

func (p *Problem) Is(target error) bool {
	if p == nil {
		return target == nil
	}
	return errors.Is(p.cause, target)
}

func (p *Problem) MarshalJSON() ([]byte, error) {
	if p == nil {
		return []byte("null"), nil
	}
	if p.data == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(p.data)
}

func (p *Problem) UnmarshalJSON(b []byte) error {
	if p == nil {
		return errors.New("problem: UnmarshalJSON on nil *Problem")
	}
	if p.data == nil {
		p.data = make(map[string]any)
	}
	for k := range p.data {
		delete(p.data, k)
	}
	return json.Unmarshal(b, &p.data)
}

func (p *Problem) WriteHeaderTo(w http.ResponseWriter) {
	w.Header().Set("Content-Type", MediaType)
	if p == nil || p.data == nil {
		return
	}
	if s, ok := p.data["status"]; ok {
		if code, ok := asStatusCode(s); ok {
			w.WriteHeader(code)
		}
	}
}

func (p *Problem) WriteTo(w http.ResponseWriter) (int, error) {
	return p.write(w, http.StatusInternalServerError)
}

func (p *Problem) write(w http.ResponseWriter, fallback int) (int, error) {
	w.Header().Set("Content-Type", MediaType)

	code := fallback
	if code == 0 {
		code = http.StatusInternalServerError
	}

	if p != nil {
		if s, ok := p.data["status"].(int); ok {
			code = s
		}
	}

	w.WriteHeader(code)
	return w.Write(p.JSON())
}

func Wrap(err error) Option {
	return optionFunc(func(p *Problem) { p.cause = err })
}

func WrapPublic(err error) Option {
	return optionFunc(func(p *Problem) {
		p.cause = err
		ensureMap(p)
		p.data["cause"] = err.Error()
	})
}

func Type(uri string) Option {
	return optionFunc(func(p *Problem) {
		ensureMap(p)
		p.data["type"] = uri
	})
}

func Title(title string) Option {
	return optionFunc(func(p *Problem) {
		ensureMap(p)
		p.data["title"] = title
	})
}

func Status(status int) Option {
	return optionFunc(func(p *Problem) {
		ensureMap(p)
		p.data["status"] = status
	})
}

func Detail(detail string) Option {
	return optionFunc(func(p *Problem) {
		ensureMap(p)
		p.data["detail"] = detail
	})
}

func Instance(uri string) Option {
	return optionFunc(func(p *Problem) {
		ensureMap(p)
		p.data["instance"] = uri
	})
}

func Ext(key string, value any) Option {
	return optionFunc(func(p *Problem) {
		ensureMap(p)
		p.data[key] = value
	})
}

func Custom(key string, value any) Option { return Ext(key, value) }

func (p *Problem) With(key string, value any) *Problem {
	return p.Append(Ext(key, value))
}

func ensureMap(p *Problem) {
	if p.data == nil {
		p.data = make(map[string]any)
	}
}

func asStatusCode(v any) (int, bool) {
	switch s := v.(type) {
	case int:
		return s, true
	default:
		return 0, false
	}
}
