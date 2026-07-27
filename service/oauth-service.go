package service

import (
	"bpl/client"
	"bpl/config"
	"bpl/repository"
	"bpl/utils"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type OauthState struct {
	Verifier string
	Timeout  int64
	User     *repository.User
	LastUrl  string
}

type OauthService interface {
	GetNewVerifier(user *repository.User, lastUrl string) (string, string)
	GetOauthProviderUrl(user *repository.User, provider repository.Provider, lastUrl string) string
	Verify(state string, code string, referrer *string, provider repository.Provider, oauthConfig oauth2.Config) (*OauthState, error)
	VerifyDiscord(state string, code string, referrer *string, oauthConfig oauth2.Config) (*OauthState, error)
	VerifyTwitch(state string, code string, referrer *string, oauthConfig oauth2.Config) (*OauthState, error)
	VerifyPoE(state string, code string, referrer *string, oauthConfig oauth2.Config) (*OauthState, error)
	GetApplicationToken(provider repository.Provider) (string, error)
	GetToken(provider repository.Provider) (token string, expiry *time.Time, err error)
	RefreshOnePoEToken() error
	RefreshPoETokensLoop(ctx context.Context, sleepDuration time.Duration)
	GetOauthConfig(provider repository.Provider) *oauth2.Config
}

type OauthServiceImpl struct {
	Config                     map[repository.Provider]*oauth2.Config
	clientConfig               map[repository.Provider]*clientcredentials.Config
	stateMap                   map[string]OauthState
	mu                         *sync.Mutex
	userService                UserService
	clientCredentialRepository repository.ClientCredentialsRepository
	oauthRepository            repository.OauthRepository
}

type DiscordUserResponse struct {
	Id       string `json:"id"`
	Username string `json:"username"`
}

type TwitchUserResponse struct {
	Aud            string `json:"aud"`
	Exp            int64  `json:"exp"`
	Iat            int64  `json:"iat"`
	Iss            string `json:"iss"`
	Sub            string `json:"sub"`
	Email          string `json:"email"`
	Email_verified bool   `json:"email_verified"`
	Picture        string `json:"picture"`
	Updated_at     string `json:"updated_at"`
}

type TwitchExtendedUserResponse struct {
	Data []struct {
		Id              string `json:"id"`
		Login           string `json:"login"`
		DisplayName     string `json:"display_name"`
		Type            string `json:"type"`
		BroadcasterType string `json:"broadcaster_type"`
		Description     string `json:"description"`
		ProfileImageUrl string `json:"profile_image_url"`
		OfflineImageUrl string `json:"offline_image_url"`
		ViewCount       int    `json:"view_count"`
		Email           string `json:"email"`
		CreatedAt       string `json:"created_at"`
	} `json:"data"`
}

func NewOauthService() OauthService {
	return &OauthServiceImpl{
		Config: map[repository.Provider]*oauth2.Config{
			repository.ProviderDiscord: {
				ClientID:     config.Env().DiscordClientID,
				ClientSecret: config.Env().DiscordClientSecret,
				Scopes:       []string{"identify"},
				Endpoint: oauth2.Endpoint{
					AuthURL:  "https://discord.com/oauth2/authorize",
					TokenURL: "https://discord.com/api/oauth2/token",
				},
				RedirectURL: "https://bpl-poe.com/auth/discord/callback",
			},
			repository.ProviderTwitch: {
				ClientID:     config.Env().TwitchClientID,
				ClientSecret: config.Env().TwitchClientSecret,
				Scopes:       []string{},
				Endpoint: oauth2.Endpoint{
					AuthURL:  "https://id.twitch.tv/oauth2/authorize",
					TokenURL: "https://id.twitch.tv/oauth2/token",
				},
				RedirectURL: "https://bpl-poe.com/auth/twitch/callback",
			},
			repository.ProviderPoE: {
				ClientID:     config.Env().POEClientID,
				ClientSecret: config.Env().POEClientSecret,
				Scopes:       []string{"account:profile", "account:characters", "account:league_accounts", "account:guild:stashes"},
				Endpoint: oauth2.Endpoint{
					AuthURL:  "https://www.pathofexile.com/oauth/authorize",
					TokenURL: "https://www.pathofexile.com/oauth/token",
				},
				RedirectURL: "https://bpl-poe.com/auth/poe/callback",
			},
		},
		clientConfig: map[repository.Provider]*clientcredentials.Config{
			repository.ProviderTwitch: {
				ClientID:     config.Env().TwitchClientID,
				ClientSecret: config.Env().TwitchClientSecret,
				TokenURL:     "https://id.twitch.tv/oauth2/token",
			},
			repository.ProviderPoE: {
				ClientID:     config.Env().POEClientID,
				ClientSecret: config.Env().POEClientSecret,
				TokenURL:     "https://www.pathofexile.com/oauth/token",
				Scopes:       []string{"service:psapi"},
			},
		},
		stateMap:                   make(map[string]OauthState),
		mu:                         &sync.Mutex{},
		userService:                NewUserService(),
		clientCredentialRepository: repository.NewClientCredentialsRepository(),
		oauthRepository:            repository.NewOauthRepository(),
	}
}

func (e *OauthServiceImpl) GetNewVerifier(user *repository.User, lastUrl string) (string, string) {
	// clean up old verifiers
	e.mu.Lock()
	defer e.mu.Unlock()
	for verifier, v := range e.stateMap {
		if v.Timeout < time.Now().Unix() {
			delete(e.stateMap, verifier)
		}
	}
	state := embedOriginInState(oauth2.GenerateVerifier(), lastUrl)
	verifier := oauth2.GenerateVerifier()
	e.stateMap[state] = OauthState{
		Verifier: verifier,
		Timeout:  time.Now().Add(30 * time.Minute).Unix(),
		User:     user,
		LastUrl:  lastUrl,
	}
	return state, verifier
}

// embedOriginInState appends the frontend origin the flow started from to
// the random oauth state, as "<randomState>.<base64url(origin)>". Providers
// only allow us to register a single redirect_uri (https://bpl-poe.com/...),
// so when we're testing against localhost (or another approved test
// domain), the oauth callback still lands on bpl-poe.com first. The state
// param is the only thing providers guarantee to echo back completely
// unchanged, so encoding the origin into it lets the bpl-poe.com callback
// page recognize this case *client-side*, with no backend round trip, and
// redirect the browser (same path, same query string, different origin) to
// the origin that actually needs to exchange the code - since that's the
// backend that holds the matching state/verifier in its stateMap.
func embedOriginInState(state string, lastUrl string) string {
	parsed, err := url.Parse(lastUrl)
	if err != nil || parsed.Host == "" {
		return state
	}
	origin := parsed.Scheme + "://" + parsed.Host
	return state + "." + base64.RawURLEncoding.EncodeToString([]byte(origin))
}

func (e *OauthServiceImpl) GetOauthConfig(provider repository.Provider) *oauth2.Config {
	return e.Config[provider]
}

// sanitizeLastUrl makes sure a lastUrl coming from an oauth redirect request
// can only ever be used to redirect the browser (and, via embedOriginInState,
// to forward the in-progress callback) to an origin we recognize. Anything
// we don't recognize is dropped in favor of a same-origin default so we
// never redirect - or hand off which backend should process the callback -
// to an arbitrary domain.
func sanitizeLastUrl(lastUrl string) string {
	parsed, err := url.Parse(lastUrl)
	if err != nil || parsed.Host == "" {
		// relative path, always same-origin and therefore safe
		return lastUrl
	}
	if config.IsApprovedOrigin(parsed.Scheme + "://" + parsed.Host) {
		return lastUrl
	}
	fmt.Printf("Rejected oauth last_url with unapproved origin: %s\n", parsed.Scheme+"://"+parsed.Host)
	return "/"
}

func (e *OauthServiceImpl) GetOauthProviderUrl(user *repository.User, provider repository.Provider, lastUrl string) string {
	state, verifier := e.GetNewVerifier(user, sanitizeLastUrl(lastUrl))
	config := e.Config[provider]
	return config.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("code_challenge", oauth2.S256ChallengeFromVerifier(verifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
}

func (e *OauthServiceImpl) Verify(state string, code string, referrer *string, provider repository.Provider, oauthConfig oauth2.Config) (*OauthState, error) {
	switch provider {
	case repository.ProviderDiscord:
		return e.VerifyDiscord(state, code, referrer, oauthConfig)
	case repository.ProviderTwitch:
		return e.VerifyTwitch(state, code, referrer, oauthConfig)
	case repository.ProviderPoE:
		return e.VerifyPoE(state, code, referrer, oauthConfig)
	default:
		return nil, fmt.Errorf("not implemented")
	}
}

func (e *OauthServiceImpl) addAccountToUser(authState *OauthState, referrer *string, accountId string, accountName string, token *oauth2.Token, provider repository.Provider) (*OauthState, error) {
	user, err := e.userService.GetUserByOauthProviderAndAccountId(provider, accountId)
	if err == nil {
		fmt.Printf("Updating %s account %s for user %s\n", provider, accountName, user.DisplayName)
		authState.User = user
	} else if authState.User == nil {
		fmt.Printf("Creating new user for %s account %s\n", provider, accountName)
		authState.User = &repository.User{
			Permissions:   []repository.Permission{},
			DisplayName:   accountName,
			OauthAccounts: []*repository.Oauth{},
			Referrer:      referrer,
		}
	} else {
		fmt.Printf("Adding %s account %s to user %s\n", provider, accountName, authState.User.DisplayName)
	}
	newAccount := &repository.Oauth{
		UserId:        authState.User.Id,
		Provider:      provider,
		AccessToken:   token.AccessToken,
		RefreshToken:  token.RefreshToken,
		AccountId:     accountId,
		Name:          accountName,
		Expiry:        token.Expiry,
		RefreshExpiry: token.Expiry,
	}
	if provider == repository.ProviderPoE {
		// PoE refresh tokens are valid for 90 days after fetching a new access token
		newAccount.RefreshExpiry = time.Now().Add(90 * 24 * time.Hour)
	}
	authState.User.OauthAccounts = append(
		utils.Filter(authState.User.OauthAccounts, func(oauthAccount *repository.Oauth) bool {
			return oauthAccount.Provider != provider
		}),
		newAccount,
	)
	err = e.oauthRepository.DeleteOauthsByUserIdAndProvider(authState.User.Id, provider)
	if err != nil {
		fmt.Printf("Failed to delete old oauth accounts: %v\n", err)
	}
	_, err = e.userService.SaveUser(authState.User)
	if err != nil {
		fmt.Printf("Failed to save user: %v\n", err)
	}
	return authState, err
}
func (e *OauthServiceImpl) fetchToken(oauthConfig oauth2.Config, state string, code string) (*OauthState, *oauth2.Token, error) {
	e.mu.Lock()
	authState, ok := e.stateMap[state]
	e.mu.Unlock()
	if !ok {
		return nil, nil, fmt.Errorf("state is unknown")
	}

	token, err := oauthConfig.Exchange(context.Background(), code, oauth2.SetAuthURLParam("code_verifier", authState.Verifier))
	if err != nil {
		return nil, nil, err
	}
	return &authState, token, nil
}

func (e *OauthServiceImpl) VerifyDiscord(state string, code string, referrer *string, oauthConfig oauth2.Config) (*OauthState, error) {

	authState, token, err := e.fetchToken(oauthConfig, state, code)
	if err != nil {
		return nil, err
	}
	client := oauthConfig.Client(context.Background(), token)
	response, err := client.Get("https://discord.com/api/users/@me")
	if err != nil {
		fmt.Printf("Failed to get discord user: %v\n", err)
		return nil, err
	}
	defer utils.Closer(response.Body)()
	discordUser := &DiscordUserResponse{}
	err = json.NewDecoder(response.Body).Decode(discordUser)
	if err != nil {
		fmt.Printf("Failed to decode discord user response: %v\n", err)
		return nil, fmt.Errorf("failed to decode discord user response: %v", err)
	}
	return e.addAccountToUser(authState, referrer, discordUser.Id, discordUser.Username, token, repository.ProviderDiscord)
}

func (e *OauthServiceImpl) VerifyTwitch(state string, code string, referrer *string, oauthConfig oauth2.Config) (*OauthState, error) {
	authState, token, err := e.fetchToken(oauthConfig, state, code)
	if err != nil {
		return nil, err
	}
	response, err := e.Config[repository.ProviderTwitch].Client(context.Background(), token).Get("https://id.twitch.tv/oauth2/userinfo")
	if err != nil {
		return nil, err
	}
	twitchUser := &TwitchUserResponse{}
	err = json.NewDecoder(response.Body).Decode(twitchUser)
	_ = response.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to decode twitch user response: %v", err)
	}
	twitchId := twitchUser.Sub

	req := &http.Request{
		URL: &url.URL{
			Scheme:   "https",
			Host:     "api.twitch.tv",
			Path:     "/helix/users",
			RawQuery: "id=" + twitchId,
		},
		Header: http.Header{
			"Authorization": {"Bearer " + token.AccessToken},
			"Client-Id":     {config.Env().TwitchClientID},
		},
	}
	client := &http.Client{}
	response, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	twitchExtendedUser := &TwitchExtendedUserResponse{}
	err = json.NewDecoder(response.Body).Decode(twitchExtendedUser)
	_ = response.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to decode twitch extended user response: %v", err)
	}
	return e.addAccountToUser(authState, referrer, twitchId, twitchExtendedUser.Data[0].DisplayName, token, repository.ProviderTwitch)
}

func (e *OauthServiceImpl) VerifyPoE(state string, code string, referrer *string, oauthConfig oauth2.Config) (*OauthState, error) {
	client := client.NewPoEClient(1, true, 10)
	e.mu.Lock()
	authState, ok := e.stateMap[state]
	e.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("state is unknown")
	}
	resp, clientError := client.GetAccessToken(oauthConfig.ClientID, oauthConfig.ClientSecret, code, authState.Verifier, oauthConfig.Scopes, oauthConfig.RedirectURL)
	if clientError != nil {
		fmt.Printf("Failed to get access token: %v\n", clientError)
		return nil, fmt.Errorf("failed to get access token: %v", clientError)
	}
	token := &oauth2.Token{
		AccessToken:  resp.AccessToken,
		TokenType:    resp.TokenType,
		RefreshToken: resp.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second),
	}
	profile, clientError := client.GetAccountProfile(token.AccessToken)
	if clientError != nil {
		fmt.Printf("Failed to get profile: %v\n", clientError)
		return nil, fmt.Errorf("failed to get profile: %v", clientError)
	}
	return e.addAccountToUser(&authState, referrer, profile.Uuid, profile.Name, token, repository.ProviderPoE)
}

func (e *OauthServiceImpl) GetApplicationToken(provider repository.Provider) (string, error) {
	credentials, err := e.clientCredentialRepository.GetClientCredentialsByName(provider)
	if err != nil || (credentials.Expiry != nil && credentials.Expiry.Before(time.Now())) {
		token, expiry, err := e.GetToken(provider)
		if err != nil {
			return "", err
		}
		if credentials == nil {
			credentials = &repository.ClientCredentials{
				Name:        provider,
				AccessToken: token,
				Expiry:      expiry,
			}
		} else {
			credentials.AccessToken = token
			credentials.Expiry = expiry
		}
		e.clientCredentialRepository.SaveClientCredentials(credentials)
	}
	return credentials.AccessToken, nil
}

func (e *OauthServiceImpl) GetToken(provider repository.Provider) (token string, expiry *time.Time, err error) {
	if provider == repository.ProviderPoE {
		poeClient := client.NewPoEClient(1, false, 10)
		tokenResponse, hhtpErr := poeClient.GetClientCredentials(config.Env().POEClientID, config.Env().POEClientSecret)
		if hhtpErr != nil {
			return "", nil, fmt.Errorf("failed to get PoE token: %s", hhtpErr.Description)
		}
		var expiry *time.Time = nil
		if tokenResponse.ExpiresIn != nil {
			x := time.Now().Add(time.Duration(*tokenResponse.ExpiresIn) * time.Second)
			expiry = &x
		}
		return tokenResponse.AccessToken, expiry, nil
	}

	config, ok := e.clientConfig[provider]
	if !ok {
		return "", nil, fmt.Errorf("provider not found")
	}
	if config.ClientID == "" || config.ClientSecret == "" {
		return "", nil, fmt.Errorf("client ID or secret not set")
	}
	oauthToken, err := config.Token(context.Background())
	if err != nil {
		return "", nil, err
	}
	return oauthToken.AccessToken, &oauthToken.Expiry, nil
}

func (e *OauthServiceImpl) RefreshOnePoEToken() error {
	oauth, err := e.oauthRepository.GetOauthForTokenRefresh(repository.ProviderPoE)
	if err != nil {
		// no token to refresh, this is not an error
		return nil
	}
	oauthConfig := e.Config[repository.ProviderPoE]
	poeClient := client.NewPoEClient(1, false, 10)
	resp, clientError := poeClient.RefreshAccessToken(oauthConfig.ClientID, oauthConfig.ClientSecret, oauth.RefreshToken)
	if clientError != nil {
		oauth.RefreshToken = ""
		_, _ = e.oauthRepository.SaveOauth(oauth)
		return fmt.Errorf("failed to refresh access token: %v", clientError)
	}
	oauth.AccessToken = resp.AccessToken
	oauth.RefreshToken = resp.RefreshToken
	newExpiry := time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)
	oauth.Expiry = newExpiry
	oauth.AccountId = resp.Sub
	oauth.Name = resp.Username
	// if access token is valid for less than 27 days this means that the refresh token cannot be used to extend it anymore
	if time.Duration(resp.ExpiresIn)*time.Second < time.Duration(27*24*time.Hour) {
		oauth.RefreshExpiry = newExpiry
	}
	_, err = e.oauthRepository.SaveOauth(oauth)
	if err == nil {
		fmt.Printf("Refreshed PoE token for user %s\n", oauth.Name)
	}
	return err
}

func (e *OauthServiceImpl) RefreshPoETokensLoop(ctx context.Context, sleepDuration time.Duration) {
	ticker := time.NewTicker(sleepDuration)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := e.RefreshOnePoEToken()
			if err != nil {
				fmt.Printf("Failed to refresh PoE token: %v\n", err)
			}
		}
	}
}
