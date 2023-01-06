package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	bazOrg = `
resource "boundary_scope" "baz" {
  scope_id                 = "global"
  auto_create_admin_role   = true
  auto_create_default_role = true
}
`

	fooBazOrg = `
resource "boundary_scope" "foobaz" {
  scope_id                 = "global"
  auto_create_admin_role   = true
  auto_create_default_role = true
}
`

	bazAuthMethod = `
resource "boundary_auth_method" "baz" {
  scope_id = boundary_scope.baz.id
  type     = "password"
}
`

	foobazAuthMethod = `
resource "boundary_auth_method" "foobaz" {
  scope_id = boundary_scope.baz.id
  type     = "password"
}
`

	baseScopePrimaryAuthMethod = `
resource "boundary_scope_primary_auth_method" "baz" {
  scope_id       = boundary_scope.baz.id
  auth_method_id = boundary_auth_method.baz.id
}
`

	updatePrimaryAuthMethod = `
resource "boundary_scope_primary_auth_method" "baz" {
  scope_id       = boundary_scope.baz.id
  auth_method_id = boundary_auth_method.foobaz.id
}
`

	updateScopeId = `
resource "boundary_scope_primary_auth_method" "baz" {
  scope_id       = boundary_scope.foobaz.id
  auth_method_id = boundary_auth_method.foobaz.id
}
`
)

func TestScopePrimaryAuthMethodCreation(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckPrimaryAuthMethodResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				Config: testConfig(url, bazOrg, bazAuthMethod, baseScopePrimaryAuthMethod),
				Check: resource.ComposeTestCheckFunc(
					testCheckScopePrimaryAuthMethodResourceExists(provider, "boundary_scope_primary_auth_method.baz"),
				),
			},
			importStep("boundary_scope_primary_auth_method.baz"),
			{
				Config: testConfig(url, bazOrg, bazAuthMethod, foobazAuthMethod, updatePrimaryAuthMethod),
				Check: resource.ComposeTestCheckFunc(
					testCheckScopePrimaryAuthMethodResourceExists(provider, "boundary_scope_primary_auth_method.baz"),
				),
			},
			importStep("boundary_scope_primary_auth_method.baz"),
			// {
			// 	Config: testConfig(url, bazOrg, bazAuthMethod, baseScopePrimaryAuthMethod, updateScopeId),
			// 	Check: resource.ComposeTestCheckFunc(
			// 		testCheckScopePrimaryAuthMethodResourceExists(provider, "boundary_scope_primary_auth_method.baz"),
			// 		testCheckOldScopePrimaryAuthMethodResourceUnset(provider, "boundary_scope.baz.id"),
			// 	),
			// },
			// importStep("boundary_scope_primary_auth_method.baz"),
		},
	})
}

func testCheckOldScopePrimaryAuthMethodResourceUnset(testProvider *schema.Provider, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		scope, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}
		scopeId := scope.Primary.ID
		if scopeId == "" {
			return fmt.Errorf("ScopeId not set: %s", resourceName)
		}

		md := testProvider.Meta().(*metaData)
		scpClient := scopes.NewClient(md.client)
		apiResponse, err := scpClient.Read(context.Background(), scopeId)
		if err != nil {
			return fmt.Errorf("Got an error when reading scope %q: %w", scopeId, err)
		}

		if apiResponse.GetItem().PrimaryAuthMethodId != "" {
			return fmt.Errorf("primary auth method was not unset for scope %s", resourceName)
		}

		return nil
	}
}

func testCheckScopePrimaryAuthMethodResourceExists(testProvider *schema.Provider, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		primaryAuthMethod, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not Found: %s", resourceName)
		}

		actualScopeId := primaryAuthMethod.Primary.ID
		if actualScopeId == "" {
			return fmt.Errorf("ScopeId not set: %s", resourceName)
		}

		actualAuthMethodId := primaryAuthMethod.Primary.Attributes[authMethodIdKey]
		if actualAuthMethodId == "" {
			return fmt.Errorf("AuthMethodId not set: %s", resourceName)
		}

		md := testProvider.Meta().(*metaData)
		scpClient := scopes.NewClient(md.client)
		apiResponse, err := scpClient.Read(context.Background(), actualScopeId)
		if err != nil {
			return fmt.Errorf("Got an error when reading scope %q: %w", actualScopeId, err)
		}

		gotAuthMethodId := apiResponse.GetItem().PrimaryAuthMethodId
		if gotAuthMethodId != actualAuthMethodId {
			return fmt.Errorf("Primary AuthMethod Id not matching expected value. got %s. wanted %s.", gotAuthMethodId, actualAuthMethodId)
		}

		return nil
	}
}

func testAccCheckPrimaryAuthMethodResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// retrieve the connection established in Provider configuration
		md := testProvider.Meta().(*metaData)
		scp := scopes.NewClient(md.client)

		for _, rs := range s.RootModule().Resources {
			id := rs.Primary.ID
			switch rs.Type {
			case "boundary_scope_primary_auth_method":
				_, err := scp.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Response().StatusCode() != http.StatusNotFound {
					return fmt.Errorf("didn't get a 404 when reading destroyed resource %q: %w", id, err)
				}
			default:
				continue
			}
		}
		return nil
	}
}