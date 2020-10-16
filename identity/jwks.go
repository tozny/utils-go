package identity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/tozny/utils-go/logging"
	"github.com/tozny/utils-go/server"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

const (
	// ClaimsKey is the context key used for getting the JWT claims from request context
	ClaimsKey ctxKey = "jwtClaims"
)

type ctxKey string

// Expected exposes the JWT expected type through this package for ease of use
type Expected = jwt.Expected

// PublicClaims exposes the JWT claims type through this package for ease of use
type PublicClaims = jwt.Claims

// Claims wraps a set of parse claims, as well as keeps the raw claims bytes
// handy so they can be decoded into different structs with ease.
type Claims struct {
	rawClaims json.RawMessage
	allClaims map[string]interface{}
}

// ClaimsFromMap creates a claims struct from an existing map
func ClaimsFromMap(allClaims map[string]interface{}) (Claims, error) {
	rawClaims, err := json.Marshal(&allClaims)
	return Claims{rawClaims, allClaims}, err
}

// ClaimsFromJSON creates a claims struct from a set of raw bytes
func ClaimsFromJSON(rawClaims []byte) (Claims, error) {
	var allClaims map[string]interface{}
	err := json.Unmarshal(rawClaims, &allClaims)
	return Claims{rawClaims, allClaims}, err
}

// MarshalClaims creates a Claims instance from a JSON serializable struct
func MarshalClaims(v interface{}) (Claims, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return Claims{}, err
	}
	return ClaimsFromJSON(b)
}

// Unmarshal takes a destination struct and decodes the claims information into it
func (c *Claims) Unmarshal(dest interface{}) error {
	return json.Unmarshal(c.rawClaims, dest)
}

// Raw provides access to the raw claims bytes
func (c *Claims) Raw() []byte {
	return c.rawClaims
}

// Get allows access to the parse map taking a string as a key and returning the
// stored interface, as well as a boolean indicating the key was or was not
// present in the claims at all.
func (c *Claims) Get(key string) (interface{}, bool) {
	res, ok := c.allClaims[key]
	return res, ok
}

// ValidatePublicClaims ensures the public claims in a token are valid, including
// timestamps included in the token within a 1 minute leeway
func (c *Claims) ValidatePublicClaims(expected Expected) error {
	var publicClaims PublicClaims
	err := c.Unmarshal(&publicClaims)
	if err != nil {
		return fmt.Errorf("unable to marshal public claims: %v", err)
	}
	return publicClaims.Validate(expected)
}

// ValidatePublicClaimsWithLeeway ensures the public claims in a token are valid,
// including timestamps included in the token within the duration specified
func (c *Claims) ValidatePublicClaimsWithLeeway(expected Expected, leeway time.Duration) error {
	var publicClaims PublicClaims
	err := c.Unmarshal(&publicClaims)
	if err != nil {
		return fmt.Errorf("unable to marshal public claims: %v", err)
	}
	return publicClaims.ValidateWithLeeway(expected, leeway)
}

// JWKS wraps management of JWKS, typically fetched from a public endpoint
type JWKS struct {
	Endpoint        string
	JWKSet          jose.JSONWebKeySet
	TimeoutInterval int
	timeout         time.Time
	logging.Logger
}

// NewJWKS sets up a new JWKS struct configured for the provided endpoint
func NewJWKS(endpoint string, timeout int, logger logging.Logger) JWKS {
	return JWKS{
		Endpoint:        endpoint,
		TimeoutInterval: timeout,
		Logger:          logger,
	}
}

// Middleware returns a middleware function which will authenticate a request with the JWK set
func (jwks *JWKS) Middleware(validateClaims func(Claims) error) server.Middleware {
	return server.MiddlewareFunc(func(h http.Handler, w http.ResponseWriter, r *http.Request) {
		bearer, err := server.ExtractBearerToken(r)
		if err != nil {
			jwks.Errorf("Failed to extract Bearer token from request: %+v", err)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		token, err := jwt.ParseSigned(bearer)
		if err != nil {
			jwks.Errorf("Failed to parse JWT: %+v", err)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		claimsMap := map[string]interface{}{}
		keys, err := jwks.Set(r.Context())
		if err != nil {
			jwks.Errorf("Failed to fetch JWK Set: %q", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if err := token.Claims(&keys, &claimsMap); err != nil {
			fmt.Printf("Invalid JWS signature on Bearer token using JWKS: %+v", keys)
			jwks.Errorf("Invalid JWS signature on Bearer token using JWKS: %+v", keys)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		claims, err := ClaimsFromMap(claimsMap)
		if err := validateClaims(claims); err != nil {
			jwks.Errorf("JWT claims failed to validate: %+v", claims)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		// The token appears valid, add the parsed claims to request context for downstream use if needed
		r = r.WithContext(context.WithValue(r.Context(), ClaimsKey, claims))
		// Send the request to the next handler
		h.ServeHTTP(w, r)
	})
}

// Set returns a JSON Web Key Set either from memory, or fetched from the endpoint
func (jwks *JWKS) Set(ctx context.Context) (jose.JSONWebKeySet, error) {
	now := time.Now()
	if jwks.timeout.IsZero() || now.After(jwks.timeout) {
		set, err := jwks.load(ctx)
		if err != nil {
			return set, fmt.Errorf("updating set: %+v", err)
		}
		jwks.JWKSet = set
		jwks.timeout = now.Add(time.Second * time.Duration(jwks.TimeoutInterval))
	}
	return jwks.JWKSet, nil
}

// Load atttempts to fetch and decode a JWKS from a JWKS endpoint
func (jwks *JWKS) load(ctx context.Context) (jose.JSONWebKeySet, error) {
	// Set up the finale result
	var result jose.JSONWebKeySet
	// Make the HTTP request with context
	request, err := http.NewRequest(http.MethodGet, jwks.Endpoint, nil)
	if err != nil {
		return result, err
	}
	client := &http.Client{}
	response, err := client.Do(request.WithContext(ctx))
	if err != nil {
		return result, fmt.Errorf("problem making JWKS request: %+v", err)
	}
	defer response.Body.Close()
	// Make sure we received a valid HTTP response and provide error context if we did not.
	if !(response.StatusCode >= 200 && response.StatusCode <= 299) {
		// At this point, throw away an error reading the body
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return result, fmt.Errorf("unexpected response status (%d), but unable to read error body: %+v", response.StatusCode, err)
		}
		return result, fmt.Errorf("unexpected response status (%d) when fetching JWKS: %+v", response.StatusCode, body)
	}
	// Read the full JSON body and decode
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return result, fmt.Errorf("unable to read JWKS body: %+v", err)
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return result, fmt.Errorf("unable to unmarshal JWKS body: %s %+v", body, err)
	}
	return result, err
}

// AuthenticatedClaims fetches the parsed claims struct out of request context if
// it is present, erroring if the claims do not appear present.
func AuthenticatedClaims(r *http.Request) (Claims, error) {
	claimsInterface := r.Context().Value(ClaimsKey)
	claims, ok := claimsInterface.(Claims)
	if !ok {
		return claims, errors.New("claims not present in request context")
	}
	return claims, nil
}
