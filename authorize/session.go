package authorize

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/pomerium/pomerium/config"
	"github.com/pomerium/pomerium/internal/encoding"
	"github.com/pomerium/pomerium/internal/sessions"
	"github.com/pomerium/pomerium/internal/sessions/cookie"
	"github.com/pomerium/pomerium/internal/sessions/header"
	"github.com/pomerium/pomerium/internal/sessions/queryparam"
	"github.com/pomerium/pomerium/internal/urlutil"
)

func loadSession(req *http.Request, options config.Options, encoder encoding.MarshalUnmarshaler) ([]byte, error) {
	var loaders []sessions.SessionLoader
	cookieStore, err := getCookieStore(options, encoder)
	if err != nil {
		return nil, err
	}
	loaders = append(loaders,
		cookieStore,
		header.NewStore(encoder, "Pomerium"),
		queryparam.NewStore(encoder, urlutil.QuerySession),
	)

	for _, loader := range loaders {
		sess, err := loader.LoadSession(req)
		if err != nil && !errors.Is(err, sessions.ErrNoSessionFound) {
			return nil, err
		} else if err == nil {
			return []byte(sess), nil
		}
	}

	return nil, sessions.ErrNoSessionFound
}

func getCookieStore(options config.Options, encoder encoding.MarshalUnmarshaler) (sessions.SessionStore, error) {
	cookieOptions := &cookie.Options{
		Name:     options.CookieName,
		Domain:   options.CookieDomain,
		Secure:   options.CookieSecure,
		HTTPOnly: options.CookieHTTPOnly,
		Expire:   options.CookieExpire,
	}
	cookieStore, err := cookie.NewStore(cookieOptions, encoder)
	if err != nil {
		return nil, err
	}
	return cookieStore, nil
}

func getJWTSetCookieHeaders(cookieStore sessions.SessionStore, rawjwt []byte) (map[string]string, error) {
	recorder := httptest.NewRecorder()
	err := cookieStore.SaveSession(recorder, nil /* unused by cookie store */, string(rawjwt))
	if err != nil {
		return nil, fmt.Errorf("authorize: error saving cookie: %w", err)
	}

	hdrs := make(map[string]string)
	for k, vs := range recorder.Header() {
		for _, v := range vs {
			hdrs[k] = v
		}
	}
	return hdrs, nil
}
