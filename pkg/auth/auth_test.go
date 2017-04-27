package auth

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/jarcoal/httpmock.v1"
)

func TestAuth(t *testing.T) {
	mockTransport := httpmock.NewMockTransport()
	client.Transport = mockTransport

	testUser := SignedInUser{
		Id:        3,
		OrgName:   "awoods Test",
		OrgSlug:   "awoodsTest",
		OrgId:     2,
		Name:      "testKey",
		Role:      ROLE_EDITOR,
		CreatedAt: time.Now(),
		key:       "foo",
	}

	Convey("When authenticating with adminKey", t, func() {
		user, err := Auth("key", "key")
		So(err, ShouldBeNil)
		So(user.Role, ShouldEqual, ROLE_ADMIN)
		So(user.OrgId, ShouldEqual, 1)
		So(user.OrgName, ShouldEqual, "Admin")
		So(user.IsAdmin, ShouldEqual, true)
		So(user.key, ShouldEqual, "key")
	})
	Convey("when authenticating with valid Key", t, func() {
		responder, err := httpmock.NewJsonResponder(200, &testUser)
		So(err, ShouldBeNil)
		mockTransport.RegisterResponder("POST", "https://grafana.net/api/api-keys/check", responder)

		user, err := Auth("key", "foo")
		So(err, ShouldBeNil)
		So(user.Role, ShouldEqual, testUser.Role)
		So(user.OrgId, ShouldEqual, testUser.OrgId)
		So(user.OrgName, ShouldEqual, testUser.OrgName)
		So(user.OrgSlug, ShouldEqual, testUser.OrgSlug)
		So(user.IsAdmin, ShouldEqual, testUser.IsAdmin)
		So(user.key, ShouldEqual, testUser.key)
		mockTransport.Reset()
	})

	Convey("When authenticating using cache", t, func() {
		cache.Set("foo", &testUser, time.Second)
		mockTransport.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
			t.Fatalf("unexpected request made. %s %s", req.Method, req.URL.String())
			return nil, nil
		})
		user, err := Auth("key", "foo")
		So(err, ShouldBeNil)
		So(user.Role, ShouldEqual, testUser.Role)
		So(user.OrgId, ShouldEqual, testUser.OrgId)
		So(user.OrgName, ShouldEqual, testUser.OrgName)
		So(user.OrgSlug, ShouldEqual, testUser.OrgSlug)
		So(user.IsAdmin, ShouldEqual, testUser.IsAdmin)
		So(user.key, ShouldEqual, testUser.key)
		mockTransport.Reset()
	})

	Convey("When authenticating with invalid org id 1", t, func() {
		cache.Clear()
		responder, err := httpmock.NewJsonResponder(200, &testUser)
		So(err, ShouldBeNil)
		mockTransport.RegisterResponder("POST", "https://grafana.net/api/api-keys/check", responder)

		originalValidOrgIds := validOrgIds
		defer func() { validOrgIds = originalValidOrgIds }()
		validOrgIds = int64SliceFlag{1}

		user, err := Auth("key", "foo")
		So(user, ShouldBeNil)
		So(err, ShouldEqual, ErrInvalidOrgId)
		mockTransport.Reset()
	})

	Convey("When authenticating with invalid org id 2", t, func() {
		cache.Clear()
		responder, err := httpmock.NewJsonResponder(200, &testUser)
		So(err, ShouldBeNil)
		mockTransport.RegisterResponder("POST", "https://grafana.net/api/api-keys/check", responder)

		originalValidOrgIds := validOrgIds
		defer func() { validOrgIds = originalValidOrgIds }()

		validOrgIds = int64SliceFlag{3, 4, 5}
		user, err := Auth("key", "foo")
		So(user, ShouldBeNil)
		So(err, ShouldEqual, ErrInvalidOrgId)
		mockTransport.Reset()
	})

	Convey("When authenticating with explicitely valid org id", t, func() {
		cache.Clear()
		responder, err := httpmock.NewJsonResponder(200, &testUser)
		So(err, ShouldBeNil)
		mockTransport.RegisterResponder("POST", "https://grafana.net/api/api-keys/check", responder)

		originalValidOrgIds := validOrgIds
		defer func() { validOrgIds = originalValidOrgIds }()

		validOrgIds = int64SliceFlag{1, 2, 3, 4}
		user, err := Auth("key", "foo")
		So(err, ShouldBeNil)
		So(user.Role, ShouldEqual, testUser.Role)
		So(user.OrgId, ShouldEqual, testUser.OrgId)
		So(user.OrgName, ShouldEqual, testUser.OrgName)
		So(user.OrgSlug, ShouldEqual, testUser.OrgSlug)
		So(user.IsAdmin, ShouldEqual, testUser.IsAdmin)
		So(user.key, ShouldEqual, testUser.key)
		mockTransport.Reset()
	})

	Convey("When authenticating using expired cache", t, func() {
		cache.Set("bar", &testUser, 0)
		responder, err := httpmock.NewJsonResponder(200, &testUser)
		So(err, ShouldBeNil)
		mockTransport.RegisterResponder("POST", "https://grafana.net/api/api-keys/check", responder)

		// make sure cached item is expired.
		cuser, valid := cache.Get("bar")
		So(cuser, ShouldNotBeNil)
		So(valid, ShouldBeFalse)

		user, err := Auth("key", "bar")
		So(err, ShouldBeNil)
		So(user.Role, ShouldEqual, testUser.Role)
		So(user.OrgId, ShouldEqual, testUser.OrgId)
		So(user.OrgName, ShouldEqual, testUser.OrgName)
		So(user.OrgSlug, ShouldEqual, testUser.OrgSlug)
		So(user.IsAdmin, ShouldEqual, testUser.IsAdmin)
		So(user.key, ShouldEqual, "bar")

		// make sure cache is now updated.
		cuser, valid = cache.Get("bar")
		So(cuser, ShouldNotBeNil)
		So(valid, ShouldBeTrue)

		mockTransport.Reset()
	})

	Convey("When authenticating using expired cache and bad g.net response", t, func() {
		cache.Set("baz", &testUser, 0)
		responder, err := httpmock.NewJsonResponder(503, nil)
		So(err, ShouldBeNil)
		mockTransport.RegisterResponder("POST", "https://grafana.net/api/api-keys/check", responder)

		// make sure cached item is expired.
		cuser, valid := cache.Get("baz")
		So(cuser, ShouldNotBeNil)
		So(valid, ShouldBeFalse)

		user, err := Auth("key", "baz")
		So(err, ShouldBeNil)
		So(user.Role, ShouldEqual, testUser.Role)
		So(user.OrgId, ShouldEqual, testUser.OrgId)
		So(user.OrgName, ShouldEqual, testUser.OrgName)
		So(user.OrgSlug, ShouldEqual, testUser.OrgSlug)
		So(user.IsAdmin, ShouldEqual, testUser.IsAdmin)
		So(user.key, ShouldEqual, testUser.key)

		// make sure cache is now updated.
		cuser, valid = cache.Get("baz")
		So(cuser, ShouldNotBeNil)
		So(valid, ShouldBeTrue)

		mockTransport.Reset()
	})
	Convey("When authenticating using expired cache and no g.net response", t, func() {
		cache.Set("baz", &testUser, 0)
		mockTransport.RegisterResponder("POST", "https://grafana.net/api/api-keys/check", func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("failed")
		})

		// make sure cached item is expired.
		cuser, valid := cache.Get("baz")
		So(cuser, ShouldNotBeNil)
		So(valid, ShouldBeFalse)

		user, err := Auth("key", "baz")
		So(err, ShouldBeNil)
		So(user.Role, ShouldEqual, testUser.Role)
		So(user.OrgId, ShouldEqual, testUser.OrgId)
		So(user.OrgName, ShouldEqual, testUser.OrgName)
		So(user.OrgSlug, ShouldEqual, testUser.OrgSlug)
		So(user.IsAdmin, ShouldEqual, testUser.IsAdmin)
		So(user.key, ShouldEqual, testUser.key)

		// make sure cache is now updated.
		cuser, valid = cache.Get("baz")
		So(cuser, ShouldNotBeNil)
		So(valid, ShouldBeTrue)

		mockTransport.Reset()
	})

}
