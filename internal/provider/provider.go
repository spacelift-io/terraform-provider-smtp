package provider

import (
	"context"
	"net/smtp"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func init() {
	schema.DescriptionKind = schema.StringMarkdown
}

func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"host": {
					Type: schema.TypeString,
					Description: `
The hostname (without port) of the SMTP server.
Can be passed using the SMTP_HOST environment variable.",
					`,
					DefaultFunc: schema.EnvDefaultFunc("SMTP_HOST", nil),
					Required:    true,
				},
				"username": {
					Type: schema.TypeString,
					Description: `
The username to use for authentication.
Can be passed using the SMTP_USERNAME environment variable.
					`,
					DefaultFunc: schema.EnvDefaultFunc("SMTP_USERNAME", nil),
					Required:    true,
				},
				"from": {
					Type: schema.TypeString,
					Description: `
The FROM value.
Can be passed using the SMTP_FROM environment variable.
					`,
					DefaultFunc: schema.EnvDefaultFunc("SMTP_FROM", ""),
					Optional:    true,
				},
				"port": {
					Type: schema.TypeInt,
					Description: `
The port of the SMTP server.
Can be passed using the SMTP_PORT environment variable.
If not set explicitly, it will default to 587.
					`,
					DefaultFunc: schema.EnvDefaultFunc("SMTP_PORT", 587),
					Optional:    true,
				},
				"cram_md5_auth": {
					Type:         schema.TypeList,
					Description:  "CRAM-MD5 authentication settings as defined in RFC 2195",
					Optional:     true,
					MaxItems:     1,
					ExactlyOneOf: []string{"cram_md5_auth", "plain_auth"},
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"secret": {
								Type: schema.TypeString,
								Description: `
The secret to use for authentication.
Can be passed using the SMTP_CRAM_MD5_SECRET environment variable.
								`,
								DefaultFunc: schema.EnvDefaultFunc("SMTP_CRAM_MD5_SECRET", nil),
								Required:    true,
								Sensitive:   true,
							},
						},
					},
				},
				"plain_auth": {
					Type:         schema.TypeList,
					Description:  "PLAIN authentication settings as defined in RFC 4616",
					Optional:     true,
					MaxItems:     1,
					ExactlyOneOf: []string{"cram_md5_auth", "plain_auth"},
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"password": {
								Type: schema.TypeString,
								Description: `
The password to use for authentication.
Can be passed using the SMTP_PLAIN_PASSWORD environment variable
								`,
								DefaultFunc: schema.EnvDefaultFunc("SMTP_PLAIN_PASSWORD", nil),
								Required:    true,
								Sensitive:   true,
							},
							"identity": {
								Type: schema.TypeString,
								Description: `
The identity to use for authentication.
Usually the identity should be the empty string, to act as username,
and this is the default.
Can be passed using the SMTP_PLAIN_IDENTITY environment variable.
								`,
								DefaultFunc: schema.EnvDefaultFunc("SMTP_PLAIN_IDENTITY", ""),
								Optional:    true,
							},
						},
					},
				},
			},
			ResourcesMap: map[string]*schema.Resource{
				"smtp_message": resourceMessage(),
			},
		}

		p.ConfigureContextFunc = configureClient

		return p
	}
}

type client struct {
	auth           smtp.Auth
	host, username string
	from           string
	port           int
}

func configureClient(ctx context.Context, r *schema.ResourceData) (interface{}, diag.Diagnostics) {
	client := &client{
		host:     r.Get("host").(string),
		port:     r.Get("port").(int),
		username: r.Get("username").(string),
		from:     r.Get("from").(string),
	}

	if cram, ok := r.GetOk("cram_md5_auth"); ok {
		cramSettings := cram.([]interface{})

		if len(cramSettings) > 0 {
			cramSettings := cramSettings[0].(map[string]interface{})
			client.auth = smtp.CRAMMD5Auth(
				client.username,
				cramSettings["secret"].(string),
			)
		}
	} else if plain, ok := r.GetOk("plain_auth"); ok {
		plainSettings := plain.([]interface{})

		if len(plainSettings) > 0 {
			plainSettings := plainSettings[0].(map[string]interface{})
			client.auth = smtp.PlainAuth(
				plainSettings["identity"].(string),
				client.username,
				plainSettings["password"].(string),
				client.host,
			)
		}
	} else {
		return nil, diag.Errorf("no authentication method specified")
	}

	return client, nil
}
