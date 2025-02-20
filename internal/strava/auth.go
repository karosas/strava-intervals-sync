package strava

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strava-intervals-description-sync/internal/strava/persistence"
)

func HandleAuthentication(w http.ResponseWriter, req *http.Request) {
	redirectUrl, err := getAuthRedirectUrl()
	if err != nil {
		log.Println("Failed to generate auth callback url", err)
	}
	http.Redirect(w, req,
		fmt.Sprintf("https://www.strava.com/oauth/authorize?client_id=%s&response_type=code&redirect_uri=%s&approval_prompt=force&scope=read,activity:read_all,activity:write",
			os.Getenv("STRAVA_CLIENT_ID"),
			url.QueryEscape(redirectUrl)), http.StatusFound)
}

func HandleAuthenticationCallback(w http.ResponseWriter, req *http.Request) {
	code := req.URL.Query().Get("code")

	if code == "" {
		log.Println("Received strava callback with no code")
		w.WriteHeader(http.StatusBadRequest)
	} else {
		if err := exchangeCodeForToken(code); err != nil {
			log.Println("Failed to exchange code", err)
		}
		if _, err := w.Write([]byte("Successfully exchanged code")); err != nil {
			log.Println("Failed to write response", err)
		}
	}
}

func RefreshToken() error {
	refreshToken, err := persistence.ReadRefreshToken()
	if err != nil {
		log.Println("Couldn't find local refresh token", err)
		return err
	}

	var buf bytes.Buffer
	log.Println("Creating token exchange request")

	writer := multipart.NewWriter(&buf)
	write := func(field, value string) {
		if err != nil {
			return
		}
		err = writer.WriteField(field, value)
	}

	write("client_id", os.Getenv("STRAVA_CLIENT_ID"))
	write("client_secret", os.Getenv("STRAVA_CLIENT_SECRET"))
	write("refresh_token", refreshToken)
	write("grant_type", "refresh_token")

	if err != nil {
		log.Println("Failed to create token exchange request body", err)
		return err
	}

	err = writer.Close()
	if err != nil {
		log.Println("Failed to close form writer", err)
		return err
	}

	return sendAndHandleTokenRequest(writer, &buf)
}

func exchangeCodeForToken(code string) error {
	var buf bytes.Buffer
	log.Println("Creating token exchange request")

	writer := multipart.NewWriter(&buf)
	var err error
	write := func(field, value string) {
		if err != nil {
			return
		}
		err = writer.WriteField(field, value)
	}

	write("client_id", os.Getenv("STRAVA_CLIENT_ID"))
	write("client_secret", os.Getenv("STRAVA_CLIENT_SECRET"))
	write("code", code)
	write("grant_type", "authorization_code")

	if err != nil {
		log.Println("Failed to create token exchange request body", err)
		return err
	}

	err = writer.Close()
	if err != nil {
		log.Println("Failed to close form writer", err)
		return err
	}

	return sendAndHandleTokenRequest(writer, &buf)
}

func sendAndHandleTokenRequest(writer *multipart.Writer, buf *bytes.Buffer) error {
	resp, err := http.Post("https://www.strava.com/oauth/token", writer.FormDataContentType(), buf)
	if err != nil {
		log.Println("Failed to send token exchange request", err)
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Printf("Unexpected token exchange status code: %v\n", resp.StatusCode)
		return errors.New("strava token exchange failed")
	}

	var authBody tokenResponse
	if err = json.NewDecoder(resp.Body).Decode(&authBody); err != nil {
		log.Println("Failed to decode token response", err)
		return err
	}

	if err = persistence.WriteAccessToken(authBody.AccessToken); err != nil {
		log.Println("Failed to write access token", err)
		return err
	}
	if err = persistence.WriteRefreshToken(authBody.RefreshToken); err != nil {
		log.Println("Failed to write refresh token", err)
		return err
	}

	return nil
}

func getAuthRedirectUrl() (string, error) {
	return url.JoinPath(os.Getenv("STRAVA_CALLBACK_BASE_URL"), AuthenticationCallbackUrl)
}
