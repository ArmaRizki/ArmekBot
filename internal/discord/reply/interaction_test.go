package reply

import (
	"errors"
	"net/http"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestIsAlreadyAcknowledged(t *testing.T) {
	resp := &http.Response{Status: "400 Bad Request"}

	ack := &discordgo.RESTError{
		Response: resp,
		Message:  &discordgo.APIErrorMessage{Code: 40060, Message: "Interaction has already been acknowledged"},
	}
	if !isAlreadyAcknowledged(ack) {
		t.Fatal("expected API code 40060 to be detected")
	}

	if isAlreadyAcknowledged(nil) {
		t.Fatal("expected a nil error to be false")
	}

	other := &discordgo.RESTError{Response: resp, Message: &discordgo.APIErrorMessage{Code: 50001}}
	if isAlreadyAcknowledged(other) {
		t.Fatal("expected another API code to be false")
	}

	if isAlreadyAcknowledged(errors.New("Interaction has already been acknowledged")) {
		t.Fatal("expected a plain error to be false")
	}
}
