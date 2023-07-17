package provider

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func recipientList(description string) *schema.Schema {
	return &schema.Schema{
		Type:         schema.TypeSet,
		Elem:         &schema.Schema{Type: schema.TypeString},
		Description:  description,
		Optional:     true,
		MinItems:     1,
		AtLeastOneOf: []string{"to", "cc", "bcc"},
		ForceNew:     true,
	}
}

func resourceMessage() *schema.Resource {
	return &schema.Resource{
		Description:   "Single SMTP message",
		CreateContext: resourceMessageCreate,
		ReadContext:   schema.NoopContext,
		DeleteContext: func(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
			return diag.FromErr(schema.RemoveFromState(d, nil))
		},

		Schema: map[string]*schema.Schema{
			"subject": {
				Type:        schema.TypeString,
				Description: "Subject of the message",
				Required:    true,
				ForceNew:    true,
			},
			"body": {
				Type:        schema.TypeString,
				Description: "Body of the message",
				Required:    true,
				ForceNew:    true,
			},
			"from": {
				Type:        schema.TypeString,
				Description: "From field",
				Optional:    true,
				ForceNew:    true,
			},
			"to":  recipientList("Direct recipients of the message"),
			"cc":  recipientList("CC recipients of the message"),
			"bcc": recipientList("BCC recipients of the message"),
			"headers": {
				Type:        schema.TypeMap,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Extra headers of the message",
				Optional:    true,
				ForceNew:    true,
			},
		},
	}
}

func resourceMessageCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client)

	buffer := bytes.NewBuffer(nil)
	sumWriter := sha256.New()
	msgWriter := textproto.NewWriter(bufio.NewWriter(buffer)).DotWriter()
	writer := io.MultiWriter(sumWriter, msgWriter)

	from := client.username
	if client.from != "" {
		from = client.from
	}
	if d.Get("from") != nil && d.Get("from").(string) != "" {
		from = d.Get("from").(string)
	}

	if _, err := fmt.Fprintln(writer, "From: ", from); err != nil {
		return diag.Errorf("failed to write the From header: %v", err)
	}

	if _, err := fmt.Fprintln(writer, "Subject: ", d.Get("subject")); err != nil {
		return diag.Errorf("failed to write the Subject header: %v", err)
	}

	to := asStringList(d.Get("to").(*schema.Set).List())
	if len(to) > 0 {
		if _, err := fmt.Fprintln(writer, "To: ", strings.Join(to, ", ")); err != nil {
			return diag.Errorf("failed to write the To header: %v", err)
		}
	}

	cc := asStringList(d.Get("cc").(*schema.Set).List())
	if len(cc) > 0 {
		if _, err := fmt.Fprintln(writer, "Cc: ", strings.Join(cc, ", ")); err != nil {
			return diag.Errorf("failed to write the Cc header: %v", err)
		}
	}

	for k, v := range d.Get("headers").(map[string]interface{}) {
		if _, err := fmt.Fprintln(writer, k, ": ", v); err != nil {
			return diag.Errorf("failed to write the %s header: %v", k, err)
		}
	}

	// Write the body
	if _, err := fmt.Fprintln(writer); err != nil {
		return diag.Errorf("failed to write the body separator: %v", err)
	}

	if _, err := fmt.Fprint(writer, d.Get("body")); err != nil {
		return diag.Errorf("failed to write the body: %v", err)
	}

	if err := msgWriter.Close(); err != nil {
		return diag.Errorf("failed to close the message writer: %v", err)
	}

	// Calculate the SHA256 hash of the message
	host := fmt.Sprintf("%s:%d", client.host, client.port)

	// Define the recipients of the message.
	recipients := uniqueRecipients(to, cc, asStringList(d.Get("bcc").(*schema.Set).List()))

	if err := smtp.SendMail(host, client.auth, client.username, recipients, buffer.Bytes()); err != nil {
		return diag.Errorf("Error sending message as %s: %s", client.username, err)
	}

	d.SetId(fmt.Sprintf("%d-%x", time.Now().UnixNano(), sumWriter.Sum(nil)))

	return nil
}

func asStringList(in []interface{}) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = v.(string)
	}
	return out
}

func uniqueRecipients(recipients ...[]string) []string {
	recipientSet := make(map[string]struct{})
	for _, recipient := range recipients {
		for _, r := range recipient {
			recipientSet[r] = struct{}{}
		}
	}

	out := make([]string, 0, len(recipientSet))
	for r := range recipientSet {
		out = append(out, r)
	}

	return out
}
