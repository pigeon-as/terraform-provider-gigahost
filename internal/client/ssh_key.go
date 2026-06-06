package client

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"strings"
)

type SSHKey struct {
	KeyID    string `json:"key_id"`
	KeyName  string `json:"key_name"`
	KeyAdded string `json:"key_added"`
	KeyData  string `json:"key_data"`
}

type createSSHKeyRequest struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

func (c *Client) ListSSHKeys(ctx context.Context) ([]SSHKey, error) {
	account, err := c.GetAccount(ctx)
	if err != nil {
		return nil, err
	}
	return account.SSHKeys, nil
}

func (c *Client) GetSSHKey(ctx context.Context, id string) (*SSHKey, error) {
	keys, err := c.ListSSHKeys(ctx)
	if err != nil {
		return nil, err
	}
	for i := range keys {
		if keys[i].KeyID == id {
			return &keys[i], nil
		}
	}
	return nil, ErrNotFound
}

func (c *Client) CreateSSHKey(ctx context.Context, name, data string) (*SSHKey, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "account/sshkey", nil, createSSHKeyRequest{Name: name, Data: data})
	if err != nil {
		return nil, err
	}
	if err := c.sendRequest(req, nil); err != nil {
		return nil, err
	}

	keys, err := c.ListSSHKeys(ctx)
	if err != nil {
		return nil, err
	}
	for i := range keys {
		if strings.TrimSpace(keys[i].KeyData) == strings.TrimSpace(data) {
			return &keys[i], nil
		}
	}
	return nil, fmt.Errorf("ssh key %q was created but could not be found on the account", name)
}

func (c *Client) DeleteSSHKey(ctx context.Context, id string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, path.Join("account", "sshkey", id), nil, nil)
	if err != nil {
		return err
	}
	return c.sendRequest(req, nil)
}
