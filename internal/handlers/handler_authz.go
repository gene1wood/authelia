package handlers

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/authelia/authelia/v4/internal/authentication"
	"github.com/authelia/authelia/v4/internal/authorization"
	"github.com/authelia/authelia/v4/internal/middlewares"
	"github.com/authelia/authelia/v4/internal/utils"
)

// Handler is the middlewares.RequestHandler for Authz.
func (authz *Authz) Handler(ctx *middlewares.AutheliaCtx) {
	var (
		object    authorization.Object
		portalURL *url.URL
		err       error
	)

	if object, err = authz.handleGetObject(ctx); err != nil {
		// TODO: Adjust.
		ctx.Logger.Errorf("Error getting object: %v", err)

		ctx.ReplyUnauthorized()

		return
	}

	if !utils.IsURISecure(object.URL) {
		ctx.Logger.Errorf("Target URL '%s' has an insecure scheme '%s', only the 'https' and 'wss' schemes are supported so session cookies can be transmitted securely", object.URL.String(), object.URL.Scheme)

		ctx.ReplyUnauthorized()

		return
	}

	if portalURL, err = authz.getPortalURL(ctx, &object); err != nil {
		ctx.Logger.Errorf("Target URL '%s' does not appear to be a protected domain: %+v", object.URL.String(), err)

		ctx.ReplyUnauthorized()

		return
	}

	var (
		authn         Authn
		authenticator AuthnStrategy
	)

	if authn, authenticator, err = authz.authn(ctx); err != nil {
		// TODO: Adjust.
		ctx.Logger.Errorf("LOG ME: Target URL '%s' does not appear to be a protected domain: %+v", object.URL.String(), err)

		ctx.ReplyUnauthorized()

		return
	}

	authn.Object = object
	authn.Method = friendlyMethod(authn.Object.Method)

	ruleHasSubject, required := ctx.Providers.Authorizer.GetRequiredLevel(
		authorization.Subject{
			Username: authn.Details.Username,
			Groups:   authn.Details.Groups,
			IP:       ctx.RemoteIP(),
		},
		object,
	)

	switch isAuthzResult(authn.Level, required, ruleHasSubject) {
	case AuthzResultForbidden:
		ctx.Logger.Infof("Access to '%s' is forbidden to user %s", object.URL.String(), authn.Username)
		ctx.ReplyForbidden()
	case AuthzResultUnauthorized:
		var handler HandlerAuthzUnauthorized

		if authenticator != nil {
			handler = authenticator.HandleUnauthorized
		} else {
			handler = authz.handleUnauthorized
		}

		handler(ctx, &authn, authz.getRedirectionURL(&object, portalURL))
	case AuthzResultAuthorized:
		authz.handleAuthorized(ctx, &authn)
	}
}

func (authz *Authz) getPortalURL(ctx *middlewares.AutheliaCtx, object *authorization.Object) (portalURL *url.URL, err error) {
	if len(authz.config.Domains) == 1 {
		portalURL = authz.config.Domains[0].PortalURL

		if portalURL == nil && authz.handleGetPortalURL != nil {
			if portalURL, err = authz.handleGetPortalURL(ctx); err != nil {
				return nil, err
			}
		}

		if portalURL == nil {
			return nil, nil
		}

		if strings.HasSuffix(object.Domain, authz.config.Domains[0].Name) {
			return portalURL, nil
		}
	} else {
		for i := 0; i < len(authz.config.Domains); i++ {
			if authz.config.Domains[i].Name != "" && strings.HasSuffix(object.Domain, authz.config.Domains[i].Name) {
				return authz.config.Domains[i].PortalURL, nil
			}
		}
	}

	return nil, fmt.Errorf("the url '%s' doesn't appear to be on a protected domain", object.URL.String())
}

func (authz *Authz) getRedirectionURL(object *authorization.Object, portalURL *url.URL) (redirectionURL *url.URL) {
	if portalURL == nil {
		return nil
	}

	redirectionURL, _ = url.ParseRequestURI(portalURL.String())

	qry := redirectionURL.Query()

	qry.Set(queryArgRD, object.URL.String())

	if object.Method != "" {
		qry.Set(queryArgRM, object.Method)
	}

	redirectionURL.RawQuery = qry.Encode()

	return redirectionURL
}

func (authz *Authz) authn(ctx *middlewares.AutheliaCtx) (authn Authn, strategy AuthnStrategy, err error) {
	for _, strategy = range authz.strategies {
		if authn, err = strategy.Get(ctx); err != nil {
			if strategy.CanHandleUnauthorized() {
				return Authn{Type: authn.Type, Level: authentication.NotAuthenticated}, strategy, nil
			}

			return Authn{Type: authn.Type, Level: authentication.NotAuthenticated}, nil, err
		}

		if authn.Level != authentication.NotAuthenticated {
			break
		}
	}

	if strategy.CanHandleUnauthorized() {
		return authn, strategy, err
	}

	return authn, nil, nil
}