package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/authmethods"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	// Password auth method keys
	authmethodTypePassword          = "password"
	authmethodMinLoginNameLengthKey = "min_login_name_length"
	authmethodMinPasswordLengthKey  = "min_password_length"

	// OIDC auth method keys
	authmethodTypeOidc                              = "oidc"
	authmethodOidcStateKey                          = "state"
	authmethodOidcDiscoveryUrlKey                   = "discovery_url"
	authmethodOidcClientIdKey                       = "client_id"
	authmethodOidcClientSecretKey                   = "client_secret"
	authmethodOidcClientSecretHmacKey               = "client_secret_hmac"
	authmethodOidcMaxAgeKey                         = "max_age"
	authmethodOidcSigningAlgorithmsKey              = "signing_algorithms"
	authmethodOidcApiUrlPrefixKey                   = "api_url_prefix"
	authmethodOidcCallbackUrlKey                    = "callback_url"
	authmethodOidcCertificatesKey                   = "certificates"
	authmethodOidcAllowedAudiencesKey               = "allowed_audiences"
	authmethodOidcOverrideOidcDiscoveryUrlConfigKey = "override_oidc_discovery_url_config"
)

func resourceAuthMethod() *schema.Resource {
	return &schema.Resource{
		Description: "The auth method resource allows you to configure a Boundary auth_method.",

		CreateContext: resourceAuthMethodCreate,
		ReadContext:   resourceAuthMethodRead,
		UpdateContext: resourceAuthMethodUpdate,
		DeleteContext: resourceAuthMethodDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the account.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The auth method name. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The auth method description.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			ScopeIdKey: {
				Description: "The scope ID.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			TypeKey: {
				Description: "The resource type.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			authmethodTypePassword: {
				Type: schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						authmethodMinLoginNameLengthKey: {
							Description: "The minimum login name length.",
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
						},
						authmethodMinPasswordLengthKey: {
							Description: "The minimum password length.",
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
			authmethodTypeOidc: {
				Type: schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						authmethodOidcStateKey: {
							Description: "OIDC state",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						authmethodOidcDiscoveryUrlKey: {
							Description: "OIDC discovery URL",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						authmethodOidcClientIdKey: {
							Description: "OIDC client ID",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						authmethodOidcClientSecretKey: {
							Description: "OIDC client secret",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						authmethodOidcClientSecretHmacKey: {
							Description: "OIDC client secret HMAC",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						authmethodOidcMaxAgeKey: {
							Description: "OIDC max age",
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
						},
						authmethodOidcSigningAlgorithmsKey: {
							Description: "OIDC signing algorithms",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						authmethodOidcApiUrlPrefixKey: {
							Description: "OIDC API URL prefix",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						authmethodOidcCallbackUrlKey: {
							Description: "OIDC callback URL",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						authmethodOidcCertificatesKey: {
							Description: "OIDC certificates",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						authmethodOidcAllowedAudiencesKey: {
							Description: "OIDC allowed audiences",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						authmethodOidcOverrideOidcDiscoveryUrlConfigKey: {
							Description: "OIDC discovery URL override configuration",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func setFromAuthMethodResponseMap(d *schema.ResourceData, raw map[string]interface{}) error {
	if err := d.Set(NameKey, raw["name"]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw["description"]); err != nil {
		return err
	}
	if err := d.Set(ScopeIdKey, raw["scope_id"]); err != nil {
		return err
	}
	if err := d.Set(TypeKey, raw["type"]); err != nil {
		return err
	}

	switch raw[TypeKey].(string) {
	case authmethodTypePassword:
		if attrsVal, ok := raw["attributes"]; ok {
			attrs := attrsVal.(map[string]interface{})

			minLoginNameLength := attrs[authmethodMinLoginNameLengthKey].(json.Number)
			minLoginNameLengthInt, _ := minLoginNameLength.Int64()
			if err := d.Set(authmethodMinLoginNameLengthKey, int(minLoginNameLengthInt)); err != nil {
				return err
			}

			minPasswordLength := attrs[authmethodMinPasswordLengthKey].(json.Number)
			minPasswordLengthInt, _ := minPasswordLength.Int64()
			if err := d.Set(authmethodMinPasswordLengthKey, int(minPasswordLengthInt)); err != nil {
				return err
			}
		}

	case authmethodTypeOidc:
		if attrsVal, ok := raw["attributes"]; ok {
			attrs := attrsVal.(map[string]interface{})

			// these are always set
			d.Set(authmethodOidcStateKey, attrs[authmethodOidcStateKey].(string))
			d.Set(authmethodOidcIssuerKey, attrs[authmethodOidcIssuerKey].(string))
			d.Set(authmethodOidcClientIdKey, attrs[authmethodOidcClientIdKey].(string))
			d.Set(authmethodOidcClientSecretHmacKey, attrs[authmethodOidcClientSecretHmacKey].(string))

			// TODO(malnick) - the API can return a value with an extra newline to the top
			// of values that are in string arrays, this is the workaround. Simiarly, there
			// is a workaround in tests when comparing API state
			stripC := []string{}
			for _, cert := range attrs[authmethodOidcCaCertificatesKey].([]interface{}) {
				stripC = append(stripC, strings.TrimSpace(cert.(string)))
			}
			d.Set(authmethodOidcCaCertificatesKey, stripC)

			stripA := []string{}
			for _, aud := range attrs[authmethodOidcAllowedAudiencesKey].([]interface{}) {
				stripA = append(stripA, strings.TrimSpace(aud.(string)))
			}
			d.Set(authmethodOidcAllowedAudiencesKey, stripA)

			fmt.Printf("ca certs: %s\n", d.Get(authmethodOidcCaCertificatesKey))

			// TODO(malnick) remove after testing
			/*
				strArys := []string{authmethodOidcCaCertificatesKey, authmethodOidcAllowedAudiencesKey}

				for _, k := range strArys {
					kAry := []string{}
					for _, val := range attrs[k].([]interface{}) {
						kAry = append(kAry, val.(string))
					}
					d.Set(k, kAry)
					fmt.Printf("%s: %s\n", k, d.Get(k))
				}
			*/

			maxAge := attrs[authmethodOidcMaxAgeKey].(json.Number)
			maxAgeInt, _ := maxAge.Int64()
			d.Set(authmethodOidcMaxAgeKey, maxAgeInt)

			// these are set sometimes
			sometimesString := []string{
				authmethodOidcApiUrlPrefixKey,
				authmethodOidcCallbackUrlKey,
				authmethodOidcDisableDiscoveredConfigValidationKey}

			for _, k := range sometimesString {
				if val, ok := attrs[k]; ok {
					d.Set(k, val.(string))
				}
			}

			if val, ok := attrs[authmethodOidcSigningAlgorithmsKey]; ok {
				d.Set(authmethodOidcSigningAlgorithmsKey, val.([]interface{}))
			}
		}

	default:
		return errorInvalidAuthMethodType
	}

	d.SetId(raw["id"].(string))
	return nil
}

func resourceAuthMethodCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var typeStr string
	if typeVal, ok := d.GetOk(TypeKey); ok {
		typeStr = typeVal.(string)
	} else {
		return diag.Errorf("no type provided")
	}

	opts := []authmethods.Option{}
	switch typeStr {
	case authmethodTypePassword:
		var minLoginNameLength *int
		if minLengthVal, ok := d.GetOk(authmethodMinLoginNameLengthKey); ok {
			minLength := minLengthVal.(int)
			minLoginNameLength = &minLength
		}
		if minLoginNameLength != nil {
			opts = append(opts, authmethods.WithPasswordAuthMethodMinLoginNameLength(uint32(*minLoginNameLength)))
		}

		var minPasswordLength *int
		if minLengthVal, ok := d.GetOk(authmethodMinPasswordLengthKey); ok {
			minLength := minLengthVal.(int)
			minPasswordLength = &minLength
		}
		if minPasswordLength != nil {
			opts = append(opts, authmethods.WithPasswordAuthMethodMinPasswordLength(uint32(*minPasswordLength)))
		}

	case authmethodTypeOidc:
		if issuer, ok := d.GetOk(authmethodOidcIssuerKey); ok {
			opts = append(opts, authmethods.WithOidcAuthMethodIssuer(issuer.(string)))
		}
		if clientId, ok := d.GetOk(authmethodOidcClientIdKey); ok {
			opts = append(opts, authmethods.WithOidcAuthMethodClientId(clientId.(string)))
		}
		if clientSecret, ok := d.GetOk(authmethodOidcClientSecretKey); ok {
			opts = append(opts, authmethods.WithOidcAuthMethodClientSecret(clientSecret.(string)))
		}
		if maxAge, ok := d.GetOk(authmethodOidcMaxAgeKey); ok {
			opts = append(opts, authmethods.WithOidcAuthMethodMaxAge(uint32(maxAge.(int))))
		}
		if prefix, ok := d.GetOk(authmethodOidcApiUrlPrefixKey); ok {
			opts = append(opts, authmethods.WithOidcAuthMethodApiUrlPrefix(prefix.(string)))
		}
		if certs, ok := d.GetOk(authmethodOidcCaCertificatesKey); ok {
			certList := []string{}
			for _, c := range certs.([]interface{}) {
				certList = append(certList, strings.TrimSpace(c.(string)))
			}

			opts = append(opts, authmethods.WithOidcAuthMethodIdpCaCerts(certList))
		}
		if aud, ok := d.GetOk(authmethodOidcAllowedAudiencesKey); ok {
			audList := []string{}
			for _, c := range aud.([]interface{}) {
				audList = append(audList, c.(string))
			}
			opts = append(opts, authmethods.WithOidcAuthMethodAllowedAudiences(audList))
		}
		if dis, ok := d.GetOk(authmethodOidcDisableDiscoveredConfigValidationKey); ok {
			opts = append(opts, authmethods.WithOidcAuthMethodDisableDiscoveredConfigValidation(dis.(bool)))
		}
		if algos, ok := d.GetOk(authmethodOidcSigningAlgorithmsKey); ok {
			algoList := []string{}
			for _, c := range algos.([]interface{}) {
				algoList = append(algoList, c.(string))
			}
			opts = append(opts, authmethods.WithOidcAuthMethodSigningAlgorithms(algoList))
		}

	default:
		return errorInvalidAuthMethodType
	}

	nameVal, ok := d.GetOk(NameKey)
	if ok {
		nameStr := nameVal.(string)
		opts = append(opts, authmethods.WithName(nameStr))
	}

	descVal, ok := d.GetOk(DescriptionKey)
	if ok {
		descStr := descVal.(string)
		opts = append(opts, authmethods.WithDescription(descStr))
	}

	var scopeId string
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	amClient := authmethods.NewClient(md.client)

	amcr, err := amClient.Create(ctx, typeStr, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error creating auth method: %v", err)
	}
	if amcr == nil {
		return diag.Errorf("nil auth method after create")
	}

	if err := setFromAuthMethodResponseMap(d, amcr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceAuthMethodRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	amClient := authmethods.NewClient(md.client)

	amrr, err := amClient.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading auth method: %v", err)
	}
	if amrr == nil {
		return diag.Errorf("auth method nil after read")
	}

	if err := setFromAuthMethodResponseMap(d, amrr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceAuthMethodUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	amClient := authmethods.NewClient(md.client)

	opts := []authmethods.Option{}

	if d.HasChange(NameKey) {
		opts = append(opts, authmethods.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			opts = append(opts, authmethods.WithName(nameVal.(string)))
		}
	}

	if d.HasChange(DescriptionKey) {
		opts = append(opts, authmethods.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			opts = append(opts, authmethods.WithDescription(descVal.(string)))
		}
	}

	typeStr := d.Get(TypeKey).(string)
	switch typeStr {
	case authmethodTypePassword:
		if d.HasChange(authmethodMinLoginNameLengthKey) {
			opts = append(opts, authmethods.DefaultPasswordAuthMethodMinLoginNameLength())
			minLengthVal, ok := d.GetOk(authmethodMinLoginNameLengthKey)
			if ok {
				opts = append(opts, authmethods.WithPasswordAuthMethodMinLoginNameLength(uint32(minLengthVal.(int))))
			}
		}

		if d.HasChange(authmethodMinPasswordLengthKey) {
			opts = append(opts, authmethods.DefaultPasswordAuthMethodMinPasswordLength())
			minLengthVal, ok := d.GetOk(authmethodMinPasswordLengthKey)
			if ok {
				opts = append(opts, authmethods.WithPasswordAuthMethodMinPasswordLength(uint32(minLengthVal.(int))))
			}
		}

	case authmethodTypeOidc:
		if d.HasChange(authmethodOidcIssuerKey) {
			if issuer, ok := d.GetOk(authmethodOidcIssuerKey); ok {
				opts = append(opts, authmethods.WithOidcAuthMethodIssuer(issuer.(string)))
			}
		}
		if d.HasChange(authmethodOidcClientIdKey) {
			if clientId, ok := d.GetOk(authmethodOidcClientIdKey); ok {
				opts = append(opts, authmethods.WithOidcAuthMethodClientId(clientId.(string)))
			}
		}
		if d.HasChange(authmethodOidcClientSecretKey) {
			if clientSecret, ok := d.GetOk(authmethodOidcClientSecretKey); ok {
				opts = append(opts, authmethods.WithOidcAuthMethodClientSecret(clientSecret.(string)))
			}
		}
		if d.HasChange(authmethodOidcMaxAgeKey) {
			if maxAge, ok := d.GetOk(authmethodOidcMaxAgeKey); ok {
				opts = append(opts, authmethods.WithOidcAuthMethodMaxAge(uint32(maxAge.(int))))
			}
		}
		if d.HasChange(authmethodOidcSigningAlgorithmsKey) {
			if algos, ok := d.GetOk(authmethodOidcSigningAlgorithmsKey); ok {
				opts = append(opts, authmethods.WithOidcAuthMethodSigningAlgorithms(algos.([]string)))
			}
		}
		if d.HasChange(authmethodOidcApiUrlPrefixKey) {
			if prefix, ok := d.GetOk(authmethodOidcApiUrlPrefixKey); ok {
				opts = append(opts, authmethods.WithOidcAuthMethodApiUrlPrefix(prefix.(string)))
			}
		}
		if d.HasChange(authmethodOidcClientSecretHmacKey) {
			if sec, ok := d.GetOk(authmethodOidcClientSecretHmacKey); ok {
				opts = append(opts, authmethods.WithOidcAuthMethodClientSecret(sec.(string)))
			}
		}
		if d.HasChange(authmethodOidcAllowedAudiencesKey) {
			if val, ok := d.GetOk(authmethodOidcAllowedAudiencesKey); ok {
				opts = append(opts, authmethods.WithOidcAuthMethodAllowedAudiences(val.([]string)))
			}
		}
		if d.HasChange(authmethodOidcCaCertificatesKey) {
			if val, ok := d.GetOk(authmethodOidcCaCertificatesKey); ok {
				c := []string{}
				for _, cert := range val.([]string) {
					c = append(c, strings.TrimSpace(cert))
				}
				opts = append(opts, authmethods.WithOidcAuthMethodIdpCaCerts(c))
			}
		}
		if d.HasChange(authmethodOidcDisableDiscoveredConfigValidationKey) {
			if val, ok := d.GetOk(authmethodOidcDisableDiscoveredConfigValidationKey); ok {
				opts = append(opts, authmethods.WithOidcAuthMethodDisableDiscoveredConfigValidation(val.(bool)))
			}
		}
	default:
		return errorInvalidAuthMethodType
	}

	if len(opts) > 0 {
		opts = append(opts, authmethods.WithAutomaticVersioning(true))
		amur, err := amClient.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating auth method: %v", err)
		}

	if d.HasChange(NameKey) {
		if err := d.Set(NameKey, name); err != nil {
			return diag.FromErr(err)
		}
	}
	if d.HasChange(DescriptionKey) {
		if err := d.Set(DescriptionKey, desc); err != nil {
			return diag.FromErr(err)
		}
	}
	if d.HasChange(authmethodMinLoginNameLengthKey) {
		if err := d.Set(authmethodMinLoginNameLengthKey, minLoginNameLength); err != nil {
			return diag.FromErr(err)
		}
	}
	if d.HasChange(authmethodMinPasswordLengthKey) {
		if err := d.Set(authmethodMinPasswordLengthKey, minPasswordLength); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceAuthMethodDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	amClient := authmethods.NewClient(md.client)

	_, err := amClient.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting auth method: %v", err)
	}

	return nil
}
