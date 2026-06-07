package client

import (
	"context"
	"net/http"
	"path"
)

type updateServerNameRequest struct {
	Name string `json:"name"`
}

func (c *Client) UpdateServerName(ctx context.Context, id, name string) error {
	req, err := c.newRequest(ctx, http.MethodPut, path.Join("servers", id, "name"), nil, updateServerNameRequest{Name: name})
	if err != nil {
		return err
	}
	return c.sendRequest(req, nil)
}

type cancelServerRequest struct {
	Reason           string `json:"reason"`
	EarlyTermination int64  `json:"early_termination"`
}

func (c *Client) CancelServer(ctx context.Context, id string) error {
	body := cancelServerRequest{Reason: "Destroyed by Terraform", EarlyTermination: 1}
	req, err := c.newRequest(ctx, http.MethodPost, path.Join("servers", id, "cancel"), nil, body)
	if err != nil {
		return err
	}
	return c.sendRequest(req, nil)
}
