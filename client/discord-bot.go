package client

import (
	"bpl/config"
	"bpl/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

type ServerMembers struct {
	MemberIds []*discordgo.Member `json:"members"`
}

type LocalDiscordClient struct {
	Client   *http.Client
	BaseURL  string
	ServerId string
}

func NewLocalDiscordClient() *LocalDiscordClient {
	return &LocalDiscordClient{
		Client:   &http.Client{},
		BaseURL:  config.Env().DiscordBotURL,
		ServerId: config.Env().DiscordGuildID,
	}
}

func (c *LocalDiscordClient) AssignRoles() (*http.Response, error) {
	return c.Client.Post(fmt.Sprintf("%s/%s/assign-roles", c.BaseURL, c.ServerId), "application/json", nil)
}

func (c *LocalDiscordClient) CheckForServerMemberShip(discordId string) (bool, error) {
	resp, err := c.Client.Get(fmt.Sprintf("%s/%s/members/%s", c.BaseURL, c.ServerId, discordId))
	if err != nil {
		return false, err
	}
	defer utils.Closer(resp.Body)()
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return true, nil
}

func (c *LocalDiscordClient) GetServerMembers() ([]*discordgo.Member, error) {
	resp, err := c.Client.Get(fmt.Sprintf("%s/%s/members", c.BaseURL, c.ServerId))
	if err != nil {
		return nil, err
	}
	defer utils.Closer(resp.Body)()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	members := make([]*discordgo.Member, 0)
	err = json.Unmarshal(respBody, &members)
	if err != nil {
		return nil, err
	}
	return members, nil
}
