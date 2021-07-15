package force

const (
	version   string = "1.0.0"
	userAgent string = "go-force/" + version
	jsonType  string = "application/json"

	grantType             string = "password"
	grantTypeRefreshToken string = "refresh_token"

	limitsKey          string = "limits"
	queryKey           string = "query"
	queryAllKey        string = "queryAll"
	sObjectsKey        string = "sobjects"
	sObjectKey         string = "sobject"
	sObjectDescribeKey string = "describe"

	BaseQueryString string = "SELECT %v FROM %v"

	rowTemplateKey string = "rowTemplate"
	idKey          string = "{ID}"

	resourcesUri string = "/services/data/%v"
	oauthURL     string = "/services/oauth2/token"

	testLoginURI      string = "https://login.salesforce.com"
	testVersion       string = "v36.0"
	testClientId      string = "3MVG9A2kN3Bn17hs8MIaQx1voVGy662rXlC37svtmLmt6wO_iik8Hnk3DlcYjKRvzVNGWLFlGRH1ryHwS217h"
	testClientSecret  string = "4165772184959202901"
	testUserName      string = "go-force@jalali.net"
	testPassword      string = "golangrocks3"
	testSecurityToken string = "kAlicVmti9nWRKRiWG3Zvqtte"
	testEnvironment   string = "production"

	invalidSessionErrorCode string = "INVALID_SESSION_ID"

	channelParam  string = "channel"
	clientIDParam string = "clientId"
	successParam  string = "successful"
)
