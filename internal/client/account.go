package client

import (
	"context"
	"net/http"
)

type Account struct {
	CustID           string `json:"cust_id"`
	CustName         string `json:"cust_name"`
	CustAddress      string `json:"cust_address"`
	CustZipcode      string `json:"cust_zipcode"`
	CustCity         string `json:"cust_city"`
	CustCountry      string `json:"cust_country"`
	CustPhone        string `json:"cust_phone"`
	CustEmail        string `json:"cust_email"`
	CustCompanyNo    string `json:"cust_company_no"`
	CustBillingEmail string `json:"cust_billing_email"`

	SSHKeys []SSHKey `json:"sshkeys"`
}

func (c *Client) GetAccount(ctx context.Context) (*Account, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "account", nil, nil)
	if err != nil {
		return nil, err
	}

	var account Account
	if err := c.sendRequest(req, &account); err != nil {
		return nil, err
	}
	return &account, nil
}
