package force

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
)

// SObject interface all standard and custom objects must implement. Needed for uri generation.
type SObject interface {
	APIName() string
	ExternalIdAPIName() string
}

// SObjectResponse struct received from force.com API after insert of an sobject.
type SObjectResponse struct {
	Id      string    `force:"id,omitempty"`
	Errors  APIErrors `force:"error,omitempty"` //TODO: Not sure if APIErrors is the right object
	Success bool      `force:"success,omitempty"`
}

func (forceAPI *ForceApi) DescribeSObjects() (map[string]*SObjectMetaData, error) {
	if err := forceAPI.getApiSObjects(); err != nil {
		err = tracerr.Wrap(err)
		logrus.WithFields(logrus.Fields{
			"forceAPI": forceAPI,
			"err":      err,
		}).Error("error get api sobjects")
		return nil, err
	}

	return forceAPI.apiSObjects, nil
}

func (forceApi *ForceApi) DescribeSObject(in SObject) (resp *SObjectDescription, err error) {
	// Check cache
	resp, ok := forceApi.apiSObjectDescriptions[in.APIName()]
	if !ok {
		// Attempt retrieval from api
		sObjectMetaData, ok := forceApi.apiSObjects[in.APIName()]
		if !ok {
			logrus.WithField("apiName", in.APIName()).Error("unable to find metadata")
			err = fmt.Errorf("Unable to find metadata for object: %v", in.APIName())
			return
		}

		uri := sObjectMetaData.URLs[sObjectDescribeKey]

		resp = &SObjectDescription{}
		err = forceApi.Get(uri, nil, resp)
		if err != nil {
			return
		}

		// Create Comma Separated String of All Field Names.
		// Used for SELECT * Queries.
		length := len(resp.Fields)
		if length > 0 {
			var allFields bytes.Buffer
			for index, field := range resp.Fields {
				// Field type location cannot be directly retrieved from SQL Query.
				if field.Type != "location" {
					if index > 0 && index < length {
						allFields.WriteString(", ")
					}
					allFields.WriteString(field.Name)
				}
			}

			resp.AllFields = allFields.String()
		}

		forceApi.apiSObjectDescriptions[in.APIName()] = resp
	}

	return
}

func (forceApi *ForceApi) GetSObject(id string, fields []string, out SObject) (err error) {
	uri := strings.Replace(forceApi.apiSObjects[out.APIName()].URLs[rowTemplateKey], idKey, id, 1)

	params := url.Values{}
	if len(fields) > 0 {
		params.Add("fields", strings.Join(fields, ","))
	}

	err = forceApi.Get(uri, params, out.(interface{}))
	err = tracerr.Wrap(err)
	logrus.WithFields(logrus.Fields{
		"id":      id,
		"uri":     uri,
		"param":   params,
		"sobject": out,
		"apiName": out.APIName(),
		"err":     err,
	}).Info("get sobject")

	return
}

func (forceApi *ForceApi) InsertSObject(in SObject) (resp *SObjectResponse, err error) {
	uri := forceApi.apiSObjects[in.APIName()].URLs[sObjectKey]

	resp = &SObjectResponse{}
	err = forceApi.Post(uri, nil, in.(interface{}), resp)
	err = tracerr.Wrap(err)
	logrus.WithFields(logrus.Fields{
		"uri":     uri,
		"resp":    resp,
		"sobject": in,
		"apiName": in.APIName(),
		"err":     err,
	}).Info("insert sobject")

	return
}

func (forceApi *ForceApi) UpdateSObject(id string, in SObject) (err error) {
	uri := strings.Replace(forceApi.apiSObjects[in.APIName()].URLs[rowTemplateKey], idKey, id, 1)

	err = forceApi.Patch(uri, nil, in.(interface{}), nil)
	err = tracerr.Wrap(err)
	logrus.WithFields(logrus.Fields{
		"uri":     uri,
		"id":      id,
		"sobject": in,
		"apiName": in.APIName(),
		"err":     err,
	}).Info("update sobject")

	return
}

func (forceApi *ForceApi) DeleteSObject(id string, in SObject) (err error) {
	uri := strings.Replace(forceApi.apiSObjects[in.APIName()].URLs[rowTemplateKey], idKey, id, 1)

	err = forceApi.Delete(uri, nil)
	err = tracerr.Wrap(err)
	logrus.WithFields(logrus.Fields{
		"uri":     uri,
		"sobject": in,
		"apiName": in.APIName(),
		"err":     err,
	}).Info("delete sobject")

	return
}

func (forceApi *ForceApi) GetSObjectByExternalId(id string, fields []string, out SObject) (err error) {
	uri := fmt.Sprintf("%v/%v/%v", forceApi.apiSObjects[out.APIName()].URLs[sObjectKey],
		out.ExternalIdAPIName(), id)

	params := url.Values{}
	if len(fields) > 0 {
		params.Add("fields", strings.Join(fields, ","))
	}

	err = forceApi.Get(uri, params, out.(interface{}))
	err = tracerr.Wrap(err)
	logrus.WithFields(logrus.Fields{
		"id":      id,
		"uri":     uri,
		"param":   params,
		"sobject": out,
		"apiName": out.APIName(),
		"err":     err,
	}).Info("get sobject by external id")

	return
}

func (forceApi *ForceApi) UpsertSObjectByExternalId(id string, in SObject) (resp *SObjectResponse, err error) {
	uri := fmt.Sprintf("%v/%v/%v", forceApi.apiSObjects[in.APIName()].URLs[sObjectKey],
		in.ExternalIdAPIName(), id)

	resp = &SObjectResponse{}
	err = forceApi.Patch(uri, nil, in.(interface{}), resp)
	err = tracerr.Wrap(err)
	logrus.WithFields(logrus.Fields{
		"id":      id,
		"uri":     uri,
		"resp":    resp,
		"sobject": in,
		"apiName": in.APIName(),
		"err":     err,
	}).Info("upsert sobject by external id")

	return
}

func (forceApi *ForceApi) DeleteSObjectByExternalId(id string, in SObject) (err error) {
	uri := fmt.Sprintf("%v/%v/%v", forceApi.apiSObjects[in.APIName()].URLs[sObjectKey],
		in.ExternalIdAPIName(), id)

	err = forceApi.Delete(uri, nil)
	err = tracerr.Wrap(err)
	logrus.WithFields(logrus.Fields{
		"uri":     uri,
		"id":      id,
		"sobject": in,
		"apiName": in.APIName(),
		"err":     err,
	}).Info("delete sobject by external id")

	return
}
