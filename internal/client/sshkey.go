package client

import "context"

type SSHKey struct {
	KeyID    string `json:"key_id"`
	KeyName  string `json:"key_name"`
	KeyAdded string `json:"key_added"`
	KeyData  string `json:"key_data"`
}

func (c *Client) ListSSHKeys(ctx context.Context) ([]SSHKey, error) {
	account, err := c.GetAccount(ctx)
	if err != nil {
		return nil, err
	}
	return account.SSHKeys, nil
}
