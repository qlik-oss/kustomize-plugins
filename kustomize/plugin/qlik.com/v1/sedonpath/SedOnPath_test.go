package main

import (
	"fmt"
	"sigs.k8s.io/kustomize/v3/pkg/resid"
	"strings"
	"testing"

	"github.com/qlik-oss/kustomize-plugins/kustomize/utils/loadertest"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/v3/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/v3/k8sdeps/transformer"
	"sigs.k8s.io/kustomize/v3/pkg/gvk"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/resource"
)

func TestBasicSed(t *testing.T) {
	p := plugin{
		Regex: []string{"s/hello/goodbye/g"},
	}
	result, err := p.executeSed("hello there!")
	assert.NoError(t, err)
	assert.Equal(t, "goodbye there!", result)
}

func TestSedOnPath(t *testing.T) {
	basicPluginInputResources := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: the-deployment
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: the-container-1
        image: the-image-1:1
        command: 
        - /foo
        - --port=8080
        - --bar=baz
        - --TempContentServiceUrl=http://temporary-contents:6080
      - name: the-container-2
        image: the-image-2:1
        command: 
        - /abra
        - --port=8080
        - --cadabra=bam
        - --TempContentServiceUrl=http://temporary-contents:6080
`

	enginePluginInputResources := `
apiVersion: v1
data:
  common: test
  featureFlagsUri: http://$(PREFIX)-engine-feature-flags:8080
  imageRegistry: qliktech-docker.jfrog.io
  ingressAuthUrl: http://$(PREFIX)-edge-auth.$(NAMESPACE).svc.cluster.local:8080/v1/auth
  ingressClass: qlik-nginx
  keysUri: http://$(PREFIX)-engine-keys:8080/v1/keys/qlik.api.internal
  natsStreamingClusterId: $(PREFIX)-engine-messaging-nats-streaming-cluster
  natsUri: nats://$(PREFIX)-engine-messaging-nats-client:4222
  pdsUri: http://$(PREFIX)-engine-policy-decisions:5080
  tokenAuthUri: http://$(PREFIX)-engine-edge-auth:8080/v1
kind: ConfigMap
metadata:
  labels:
    app: engine
  name: engine-configs-5mmd949kth
---
apiVersion: v1
data:
  rules.yaml: |
    - allow: |
        # If app has originAppId and user is publisher in a managed space user can republish
        resource.IsApp() and resource.UserIsPublisherInManagedSpace() and resource.originAppId == app.id and resource._actions={"republish"}
        # Professional user can create an app in a personal space or in a shared space where the user is an editor.
        resource.IsApp() and user.IsProfessional() and (resource.IsOwnedInPersonalSpace() or resource.UserIsEditorInSharedSpace()) and resource._actions={"create"}
        # A user can delete app if the user is the owner in a personal space or in a shared space where the user is producer. A tenant admin can always delete an app
        resource.IsApp() and (resource.IsOwnedInPersonalSpace() or resource.UserIsEditorInSharedSpace() or resource.UserIsFacilitatorInManagedSpace()) and resource._actions={"delete"}
        # A user can copy an app if he is copying it to a personal space, a shared space where the user is producer. Read access on the src file is implicit
        resource.IsApp() and (resource.IsOwnedInPersonalSpace() or resource.UserIsEditorInSharedSpace()) and resource._actions={"duplicate"}
        # A professional user can import an app if the user is importing to a personal space or to a shared space where the user is producer.
        resource.IsApp() and user.IsProfessional() and (resource.IsOwnedInPersonalSpace() or resource.UserIsEditorInSharedSpace()) and resource._actions={"import"}
        # The app can be opened in a personal space by the owner or shared to me
        resource.IsApp() and resource.IsPersonal() and (resource.IsOwnedByMe() or resource.IsSharedWithMe()) and resource._actions={"read"}
        # A tenant admin can open all apps and delete all apps
        resource.IsApp() and user.IsTenantAdmin() and resource._actions={"read","delete"}
        # An tenant admin can change the owner, update app atributes or export an app for personal or apps in shared space
        resource.IsApp() and user.IsTenantAdmin() and (resource.IsPersonal() or resource.IsShared()) and resource._actions={"change_owner","update","export"}
        # A user can open apps that the user has access to in a shared space
        resource.IsApp() and resource.UserIsSharedSpaceMember() and resource._actions={"read"}
        # A user can open apps that the user has access to in a managed space
        resource.IsApp() and (resource.UserIsViewerInManagedSpace() or resource.UserIsFacilitatorInManagedSpace()) and resource._actions={"read"}
        # A user can open distributed apps that the user has access to
        resource.IsApp() and resource.IsDistributed() and (resource.IsOwnedByMe() or resource.IsSharedWithMe()) and resource._actions={"read"}
        # Professional user that owns the app can edit scripts and reload the app if it's a personal space or the user is editor in the shared space
        resource.IsApp() and user.IsProfessional() and resource.IsOwnedByMe() and (resource.IsPersonal() or resource.UserIsEditorInSharedSpace()) and !resource.IsPublished() and resource._actions={"reload"}
        # Professional user can reload the app if the user is facilitator in a managed space
        resource.IsApp() and user.IsProfessional() and resource.UserIsFacilitatorInManagedSpace() and resource._actions={"update","reload"}
        # A user can update app attributes on personal apps and in a shared space as producer.
        resource.IsApp() and resource.HasPrivilege("read") and (resource.UserIsEditorInSharedSpace() or resource.IsOwnedInPersonalSpace()) and resource._actions={"update"}
        # A facilitator can change the owner of an app in a shared space.
        resource.IsApp() and resource.UserIsFacilitator() and resource._actions={"change_owner"}
        # A user can change the space on personal apps and as editor of shared apps or as facilitator in managed apps.
        resource.IsApp() and (resource.IsOwnedInPersonalSpace() or resource.UserIsEditorInSharedSpace() or resource.UserIsFacilitatorInManagedSpace()) and resource._actions={"change_space"}
        # A publisher can publish an app to a managed space.
        resource.IsApp() and resource.UserIsPublisherInManagedSpace() and resource._actions={"publish"}
        # Impersonator from the ODAG service can change owner. This rule will be removed when odag does not need to impersonate
        resource.IsApp() and user.act.sub == "odag" and resource._actions={"read","change_owner"}
        # A user can export apps that the user owns or in a shared space as producer. Only user visable (objects and data) will be exported
        resource.IsApp() and !resource.HasSectionAccess() and resource.HasPrivilege("read") and (resource.IsOwnedInPersonalSpace() or resource.UserIsEditorInSharedSpace()) and resource._actions={"export"}
        # Professional user can import an appobject if the user is importing to a personal space or to a shared space where the user is producer.
        resource.IsAppObject() and user.IsProfessional() and (resource.app.IsOwnedInPersonalSpace() or resource.app.UserIsEditorInSharedSpace()) and resource._actions={"import"}
        # In apps that the user has read access to, the user can read all published objects and his personal objects and all master items.
        resource.IsAppObject() and resource.app.HasPrivilege("read") and ((resource.IsOwnedByMe() or resource.IsPublished() or resource.IsMasterObject() or resource.IsPublicObject()) and !resource.IsScriptObject()) and resource._actions={"read"}
        # If you have access to the app you can read all published objects
        resource.IsAppObject() and resource.app.HasPrivilege("read") and resource.IsPublished() and !resource.IsScriptObject() and resource._actions={"read"}
        # A professional user can read the script in owned personal apps or as editor in a shared space or facilitator in managed space.
        resource.IsAppObject() and user.IsProfessional() and (resource.app.IsOwnedInPersonalSpace() or resource.app.UserIsEditorInSharedSpace() or resource.app.UserIsFacilitatorInManagedSpace()) and resource.IsScriptObject() and resource._actions={"read"}
        # A professional user can update the script in owned personal apps or owned apps in a shared space
        resource.IsAppObject() and user.IsProfessional() and (resource.app.IsOwnedInPersonalSpace() or (resource.app.UserIsEditorInSharedSpace() and resource.IsOwnedByMe())) and resource.IsScriptObject() and resource._actions={"update"}
        # A professional user can create any object in an unpublished app that is in a personal space or in a shared space as producer
        resource.IsAppObject() and user.IsProfessional() and resource.app.HasPrivilege("read") and !resource.app.IsPublished() and (resource.app.IsOwnedInPersonalSpace() or resource.app.UserIsEditorInSharedSpace()) and resource.IsOwnedByMe() and resource._actions={"create"}
        # A user can create a story object (stories, bookmarks and snapshot) in a personal space app shared to me
        resource.IsAppObject() and resource.app.HasPrivilege("read") and !resource.app.IsPublished() and resource.app.IsPersonal() and (resource.app.IsOwnedByMe() or resource.app.IsSharedWithMe()) and resource.IsStoryObject() and !resource.IsPublished() and resource.IsOwnedByMe() and resource._actions={"create"}
        # A user can update and delete an unpublished story object owned by me (stories, bookmarks and snapshot) in a personal app owner or shared to me
        resource.IsAppObject() and resource.app.HasPrivilege("read") and !resource.app.IsPublished() and resource.app.IsPersonal() and (resource.app.IsOwnedByMe() or resource.app.IsSharedWithMe()) and resource.IsStoryObject() and !resource.IsPublished() and resource.IsOwnedByMe() and resource._actions={"update","delete"}
        # In apps that a user has read access to, a professional user can update, delete master objects and other public objects.
        resource.IsAppObject() and user.IsProfessional() and resource.app.HasPrivilege("read") and (resource.IsMasterObject() or resource.IsPublicObject()) and (resource.app.IsOwnedInPersonalSpace() or resource.app.UserIsEditorInSharedSpace()) and resource._actions={"update","delete"}
        # In apps that a user has update access to, the user can update app properties.
        resource.IsAppObject() and resource.app.HasPrivilege("update") and resource._objecttype == "appprops" and resource._actions={"update"}
        # A professional user can update and delete any unpublished object in an unpublished app that the user owns or in a shared space as producer.
        resource.IsAppObject() and user.IsProfessional() and resource.app.HasPrivilege("read") and (resource.app.IsOwnedInPersonalSpace() or resource.app.UserIsEditorInSharedSpace()) and !resource.IsPublished() and resource.IsOwnedByMe() and resource._actions={"update","delete"}
        # A professional user can publish objects in an unpublished app that the user owns or in a shared space as producer.
        resource.IsAppObject() and user.IsProfessional() and resource.app.HasPrivilege("read") and !resource.app.IsPublished() and (resource.app.IsOwnedInPersonalSpace() or resource.app.UserIsEditorInSharedSpace()) and !resource.IsScriptObject() and resource._actions={"publish"}
        # A user can create a story object (stories, bookmarks and snapshot) in a shared space where the user is a consumer.
        resource.IsAppObject() and resource.app.HasPrivilege("read") and resource.app.UserIsSharedSpaceMember() and resource.IsStoryObject() and !resource.IsPublished() and resource._actions={"create"}
        # A user can update and delete an owned personal story object (stories, bookmarks and snapshot) in a shared space where the user is a consumer.
        resource.IsAppObject() and resource.app.HasPrivilege("read") and resource.app.UserIsSharedSpaceMember() and resource.IsStoryObject() and !resource.IsPublished()  and resource.IsOwnedByMe() and resource._actions={"update","delete"}
        # A user can duplicate objects if the user has duplicate access on the app
        resource.IsAppObject() and resource.app.HasPrivilege("duplicate") and (resource.app.IsOwnedInPersonalSpace() or resource.app.UserIsEditorInSharedSpace()) and resource._actions={"duplicate"}
        # Analyser users can create app objects of type stories, snapshot and bookmarks in managed apps
        resource.IsAppObject() and resource.app.UserIsViewerInManagedSpace() and resource.app.HasPrivilege("read") and resource.IsStoryObject() and !resource.IsPublished() and resource._actions={"create"}
        # Analyser users can update and delete owned, unpublished app objects of type stories, snapshot and bookmarks in managed apps
        resource.IsAppObject() and resource.app.UserIsViewerInManagedSpace() and resource.app.HasPrivilege("read") and resource.IsStoryObject() and !resource.IsPublished() and resource.IsOwnedByMe() and resource._actions={"update","delete"}
        # Professional users can create app objects of type sheets, stories, snapshot and bookmarks in managed apps
        resource.IsAppObject() and user.IsProfessional() and resource.app.UserIsContributorInManagedSpace() and resource.app.HasPrivilege("read") and resource.IsContentObject() and resource._actions={"create"}
        # Professional users can update and delete owned, unpublished app objects of type sheets, stories, snapshot and bookmarks in managed apps that allows self service
        resource.IsAppObject() and user.IsProfessional() and resource.app.UserIsContributorInManagedSpace() and resource.app.HasPrivilege("read") and resource.IsContentObject() and !resource.IsPublished() and resource.IsOwnedByMe() and resource._actions={"update","delete"}
        # Professional users can publish and unpublish app objects of type stories, snapshot and bookmarks in managed apps
        resource.IsAppObject() and user.IsProfessional() and resource.app.UserIsContributorInManagedSpace() and resource.app.HasPrivilege("read") and resource.IsContentObject() and resource.IsOwnedByMe() and !resource.IsApproved() and resource._actions={"publish"}
        # External services can read, import, create, update, and delete apps.
        resource.IsAppOrAppObject() and user.IsExternal() and resource._actions={"read","import","create","update","delete"}
        # A professional user shall be able to upload a QlikView app
        resource.IsQvApp() and user.IsProfessional() and resource.IsOwnedInPersonalSpace() and resource._actions={"import"}
        # External services shall have full access to QlikView apps
        resource.IsQvApp() and user.IsExternal() and resource._actions={"read","import","create","update","delete"}
        # The QlikView app can be opened in a personal space by the owner or shared to me
        resource.IsQvApp() and resource.IsPersonal() and (resource.IsOwnedByMe() or resource.IsSharedWithMe()) and resource._actions={"read"}
        # A tenant admin can open and delete all QlikView apps
        resource.IsQvApp() and user.IsTenantAdmin() and resource._actions={"read","delete"}
        # A user can open QlikView apps that the user has access to in a managed space
        resource.IsQvApp() and (resource.UserIsViewerInManagedSpace() or resource.UserIsFacilitatorInManagedSpace()) and resource._actions={"read"}
        # If you have access to the QlikView app you can read all objects
        resource.IsQvAppOrAppObject() and resource.app.HasPrivilege("read") and !resource.IsScriptObject() and resource._actions={"read"}
        # If Qlikview app is in my personal space and owned by me I can share it
        resource.IsQvApp() and resource.IsPersonal() and resource.IsOwnedByMe() and !resource.IsPublished() and resource._actions={"share"}
        # A user can update QlikView personal app attributes
        resource.IsQvApp() and resource.IsOwnedInPersonalSpace() and resource.HasPrivilege("read") and resource._actions={"update"}
        # A publisher can publish a QlikView app to a managed space.
        resource.IsQvApp() and resource.UserIsPublisherInManagedSpace() and resource._actions={"publish"}
        # A user can publish QlikView personal apps to a managed space
        resource.IsQvApp() and resource.IsOwnedInPersonalSpace() and resource._actions={"publish"}
        # On global API:s we allow read to everyone
        resource._resourcetype="node" and resource._actions={"read"}
        # On global API:s with reload access professional users should have access
        resource._resourcetype="node" and user.IsProfessional() and resource._actions={"reload"}
      deny: ""
      func: |
        # User is a professional user.
        IsProfessional() (self._provision.accesstype == "professional")
        # User is an analyzer user.
        IsAnalyzer() (self._provision.accesstype == "analyzer")
        # User is a service user (external).
        IsExternal() (self.subType == "externalClient")
        # User is a tenant administrator.
        IsTenantAdmin() (self.roles =="TenantAdmin")
        # Checks parent app privileges. Privileges on parent must currently be computed in a first pass.
        app.HasPrivilege(x) (self.app._privileges == x)
        # Checks if a privilege exists on a resource.
        HasPrivilege(x) (self._actions.Matched () = x)
        # Resource is published.
        IsPublished() (self.published == "true")
        # App has Section Access.
        HasSectionAccess() (self.hassectionaccess == "true")
        # Helper macro for detecting if a property is missing or empty string.
        MissingOrEmptyProp(prop) (self.prop.empty() or self.prop = "")
        # In managed space
        IsManaged() (self.spaceId == space.id and space.type == "managed")
        # In shared space
        IsShared() (self.spaceId == space.id and space.type == "shared")
        # In distributed from another environment
        IsDistributed() (self.MissingOrEmptyProp(spaceId) and self.IsPublished())
        # Resource is approved.
        IsApproved() (self.approved = "true")
        # App is personal, in a personal space or not in a space.
        IsPersonal() ((self.spaceId == space.id and space.type == "personal") or (self.MissingOrEmptyProp(spaceId) and !self.IsPublished()))
        # App is shared with user.
        IsSharedWithMe() ((!user.MissingOrEmptyProp(sub) and self.custom.userswithaccess = user) or (!user.MissingOrEmptyProp(userId) and self.custom.userIdsWithAccess = user.userId) or (!user.MissingOrEmptyProp(groups) and self.custom.groupswithaccess = user.groups) or (!user.MissingOrEmptyProp(groupIds) and self.custom.groupIdsWithAccess = user.groupIds))
        # Resource is owned by user.
        IsOwnedByMe() (self.owner = user or (!self.MissingOrEmptyProp(ownerId) and self.ownerId == user.userId))
        # Is app object a master item.
        IsMasterObject() (self._objecttype = { "masterobject", "dimension", "measure" })
        # Is app object a story item.
        IsStoryObject() (self._objecttype = { "story", "snapshot", "bookmark" })
        # Is app object a content item (sheet, story, snapshot, bookmark).
        IsContentObject() (self._objecttype = {"sheet", "story", "snapshot", "bookmark"})
        # Public objects created outside of sheets
        IsPublicObject() (self._objecttype = { "appprops", "colormap", "odagapplink", "loadmodel" })
        # Is it the script object.
        IsScriptObject() (self._objecttype = "app_appscript")
        # Is user an editor in the space this resource belongs to. Producers are the roles produce or facilitator (space owner is automatically a facilitator)
        UserIsEditorInSharedSpace() (self.spaceId == space.id and space.type == "shared" and (space.roles == {"producer", "facilitator"} or user.userId == space.ownerId))
        # Is is a consumer in a shared space
        UserIsConsumerInSharedSpace() (self.spaceId == space.id and space.type == "shared" and space.roles == {"consumer"})
        # Is user the owner of this personal space.
        IsOwnedInPersonalSpace() (self.IsOwnedByMe() and self.IsPersonal())
        # Is user member of a shared space that the resource belongs to.
        UserIsSharedSpaceMember() (self.spaceId == space.id and space.type == "shared" and (space.roles == {"consumer", "producer", "facilitator"} or user.userId == space.ownerId))
        # User can publish to a managed space if he has the role publisher or he is the owner of the space.
        UserIsPublisherInManagedSpace() (self.spaceId == space.id and space.type == "managed" and (space.roles == {"publisher"} or user.userId == space.ownerId))
        # Is user member of a managed space that the resource belongs to.
        UserIsViewerInManagedSpace() (self.spaceId == space.id and space.type == "managed" and (space.roles == {"consumer", "contributor", "facilitator"} or user.userId == space.ownerId))
        # Is user member of a managed space that the resource belongs to.
        UserIsFacilitatorInManagedSpace() (self.spaceId == space.id and space.type == "managed" and (space.roles == {"facilitator"} or user.userId == space.ownerId))
        # Is user member of a managed space that the resource belongs to.
        UserIsContributorInManagedSpace() (self.spaceId == space.id and space.type == "managed" and (space.roles == {"contributor", "facilitator"} or user.userId == space.ownerId))
        # Is user a facilitator on the space this resource belongs to.
        UserIsFacilitator() (self.spaceId == space.id and space.type == {"shared", "managed"} and (space.roles == {"facilitator"} or user.userId == space.ownerId))
        # Is app.
        IsApp() (self._resourcetype=={"app"})
        # Is app object.
        IsAppObject() (self._resourcetype=={"app.object"})
        # Is app or app object.
        IsAppOrAppObject() (self._resourcetype=={"app", "app.object"})
        # Is a QlikView app.
        IsQvApp() (self._resourcetype=={"qvapp"})
        # Is a QlikView app or QlikView app object.
        IsQvAppOrAppObject() (self._resourcetype=={"qvapp", "qvapp.object"})
        # Is datafile.
        IsDataFile() (self._resourcetype=={"datafile"})
kind: ConfigMap
metadata:
  labels:
    app: engine
  name: engine-engine-rules-cm
---
apiVersion: v1
data:
  something: ZWxzZQ==
kind: Secret
metadata:
  labels:
    app: engine
  name: engine-secrets-d27kc695b5
type: Opaque
---
apiVersion: v1
data:
  jwtPrivateKey: LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1JR2tBZ0VCQkRDSUg1WEU1L1R3VGh6bkkxenIvcGM4RGdzVUpLa0xrN0x3bHd4YzhPaXhzZnllMUZNdDNRR2oKQmZjcEFlUVh0WldnQndZRks0RUVBQ0toWkFOaUFBU1ErSVJkSWs1RFVWSlc4THQrVWgzV2xsZ2pHakhYUmZJKwphTkJjbU00MW5Zd1JsSHZhZFVHeDVHWnFtRmY0ZzdNdjY3RlI2b3lJdFJEbVB0dDlRT0RtMkJQeUEzYWZ1UE5wCm9QRWM2RExnTzl0dHJZVEhXT2tlY0hZL1pPYnFIMm89Ci0tLS0tRU5EIEVDIFBSSVZBVEUgS0VZLS0tLS0K
kind: Secret
metadata:
  labels:
    app: engine
  name: engine-service-jwt-secret
type: Opaque
---
apiVersion: v1
kind: Service
metadata:
  annotations:
    prometheus.io/port: "9090"
    prometheus.io/scrape: "true"
  labels:
    app: engine
    chart: engine-1.48.10
    heritage: Tiller
    release: engine
  name: engine
spec:
  ports:
  - name: engine
    port: 9076
    protocol: TCP
  selector:
    app: engine
    release: engine
  type: ClusterIP
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  labels:
    app: engine
    chart: engine-1.48.10
    heritage: Tiller
    qlik.com/default: "true"
    qlik.com/engine-deployment-id: 0df3179a-5089-450a-b23f-41cca10deafd
    release: engine
  name: engine-default
spec:
  progressDeadlineSeconds: 600
  replicas: 1
  selector:
    matchLabels:
      app: engine
      qix-engine: qix-engine
      qlik.com/default: "true"
      release: engine
  strategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
  template:
    metadata:
      labels:
        app: engine
        engine-nats-client: "true"
        metricsPort: "9090"
        qix-engine: qix-engine
        qix-engine-version: 12.460.0
        qlik.com/default: "true"
        qlik.com/engine-deployment-id: 0df3179a-5089-450a-b23f-41cca10deafd
        qlik.com/engine-type: qliksense
        release: engine
        servicePort: "9076"
    spec:
      containers:
      - args:
        - -S
        - AcceptEULA=no
        - -S
        - DocumentDirectory=/qlik/apps
        - -S
        - EnableRestartOnSessionStall=1
        - -S
        - PrometheusServicePort=9090
        - -S
        - DesktopPort=9076
        - -S
        - EnableNumericalAbbreviation=0
        - -S
        - HttpTrafficLogVerbosity=4
        - -S
        - TrafficLogVerbosity=0
        - -S
        - SystemLogVerbosity=4
        - -S
        - AuditLogVerbosity=0
        - -S
        - PerformanceLogVerbosity=0
        - -S
        - QixPerformanceLogVerbosity=0
        - -S
        - SessionLogVerbosity=4
        - -S
        - ScriptLogVerbosity=4
        - -S
        - SmartSearchQueryLogVerbosity=3
        - -S
        - SmartSearchIndexLogVerbosity=3
        - -S
        - SSEVerbosity=4
        - -S
        - EventBusLogVerbosity=4
        - -S
        - EnableExtServiceLogs=1
        - -S
        - ExternalServicesLogVerbosity=4
        - -S
        - BasePathPrefix=/api
        - -S
        - FolderConnectionInterface=0
        - -S
        - Autosave=1
        - -S
        - AutosaveInterval=5
        - -S
        - ValidateJsonWebTokens=2
        - -S
        - JWKSServiceUrl=http://engine-keys:8080/v1/keys/qlik.api.internal
        - -S
        - JWTSignPrivateKeyPath=/etc/secrets/jwtPrivateKey
        - -S
        - JWTSignPrivateKeyId=uHxp0YvnYQx-AqQgHnxgKqTk_HmZiT67B8SYOlzSgoM
        - -S
        - InternalTokenServiceUrl=http://engine-edge-auth:8080/v1
        - -S
        - EnableRenewUserToken=1
        - -S
        - EnableABAC=1
        - -S
        - Gen3=1
        - -S
        - EnableFilePolling=1
        - -S
        - PersistenceMode=2
        - -S
        - EnableAccessControlTrace=1
        - -S
        - SystemRules=/etc/config/rules.yaml
        - -S
        - LicenseServiceUrl=http://engine-licenses:9200
        - -S
        - LicenseCacheTimeoutSeconds=3600
        - -S
        - EnableSpaces=1
        - -S
        - SpacesServiceUrl=http://engine-spaces:6080
        - -S
        - EnableEncryptData=1
        - -S
        - UseEncryptionService=1
        - -S
        - EncryptionServiceUrl=http://engine-encryption:8080
        - -S
        - EnableFeatureFlagService=1
        - -S
        - FeatureFlagServiceUrl=http://engine-feature-flags:8080
        - -S
        - EnableGroupsService=1
        - -S
        - GroupsServiceUrl=http://engine-groups:8080
        - -S
        - EnableAppExport=1
        - -S
        - EnableTempContentService=1
        - -S
        - TempContentServiceUrl=http://engine-temporary-contents:6080
        - -S
        - UseSTAN=1
        - -S
        - STANUrl=nats://engine-nats-client:4222
        - -S
        - STANCluster=engine-nats-streaming-cluster
        - -S
        - STANUseToken=1
        - -S
        - STANMaxReconnect=60
        - -S
        - STANReconnectWait=2
        - -S
        - STANTimeout=10
        - -S
        - UseEventBus=1
        - -S
        - EnvironmentName="example"
        - -S
        - RegionName="example"
        - -S
        - ShutdownWait=1
        - -S
        - EnableQvwRestImport=1
        env:
        - name: PROMETHEUS_PORT
          value: "9090"
        - name: GRPC_DNS_RESOLVER
          value: native
        image: qlik-docker-qsefe.bintray.io/engine:12.460.0
        imagePullPolicy: null
        livenessProbe:
          failureThreshold: 3
          httpGet:
            path: /health
            port: 9076
          timeoutSeconds: 10
        name: engine
        ports:
        - containerPort: 9076
        - containerPort: 9090
          name: metrics
        readinessProbe:
          httpGet:
            path: /health
            port: 9076
        resources: {}
        volumeMounts:
        - mountPath: /qlik/apps
          name: apps-storage
        - mountPath: /etc/secrets
          name: secrets-service2service
          readOnly: true
        - mountPath: /etc/config
          name: rules-volume
        - mountPath: /home/engine/Qlik/Sense
          name: storagepath
        - mountPath: /tmp
          name: tmpdir
      dnsConfig: null
      imagePullSecrets:
      - name: artifactory-docker-secret
      options:
      - name: timeout
        value: "1"
      - name: single-request-reopen
      terminationGracePeriodSeconds: 30
      volumes:
      - name: apps-storage
        persistentVolumeClaim:
          claimName: engine
      - name: secrets-service2service
        secret:
          secretName: engine-service-jwt-secret
      - configMap:
          defaultMode: 493
          name: engine-prestop-hook
          optional: true
        name: engine-prestop-hook
      - configMap:
          name: engine-engine-rules-cm
        name: rules-volume
      - emptyDir: {}
        name: storagepath
      - emptyDir: {}
        name: tmpdir
---
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: nginx
    nginx.ingress.kubernetes.io/auth-response-headers: Authorization
    nginx.ingress.kubernetes.io/auth-url: http://engine-edge-auth.$(NAMESPACE).svc.cluster.local:8080/v1/auth
    nginx.ingress.kubernetes.io/configuration-snippet: |
      rewrite (?i)/api/(.*) /$1 break;
    nginx.ingress.kubernetes.io/proxy-body-size: 500m
    nginx.ingress.kubernetes.io/proxy-read-timeout: "600"
    nginx.org/client-max-body-size: 500m
  labels:
    app: engine
    chart: engine-1.48.10
    heritage: Tiller
    release: engine
  name: engine
spec:
  rules:
  - http:
      paths:
      - backend:
          serviceName: engine
          servicePort: 9076
        path: /api/v1/apps/import
      - backend:
          serviceName: engine
          servicePort: 9076
        path: /api/v1/apps
      - backend:
          serviceName: engine
          servicePort: 9076
        path: /api/engine/openapi
      - backend:
          serviceName: engine
          servicePort: 9076
        path: /api/engine/openapi/rpc
      - backend:
          serviceName: engine
          servicePort: 9076
        path: /api/engine/asyncapi
---
apiVersion: qixmanager.qlik.com/v1
kind: Engine
metadata:
  labels:
    app: engine
    chart: engine-1.48.10
    heritage: Tiller
    release: engine
  name: engine-reload
spec:
  metadata:
    labels:
      engine-nats-client: "true"
      qix-engine-version: 12.460.0
  podSpec:
    containers:
    - args:
      - -S
      - AcceptEULA=no
      - -S
      - DocumentDirectory=/qlik/apps
      - -S
      - EnableRestartOnSessionStall=1
      - -S
      - PrometheusServicePort=9090
      - -S
      - DesktopPort=9076
      - -S
      - EnableNumericalAbbreviation=0
      - -S
      - HttpTrafficLogVerbosity=4
      - -S
      - TrafficLogVerbosity=0
      - -S
      - SystemLogVerbosity=4
      - -S
      - AuditLogVerbosity=0
      - -S
      - PerformanceLogVerbosity=0
      - -S
      - QixPerformanceLogVerbosity=0
      - -S
      - SessionLogVerbosity=4
      - -S
      - ScriptLogVerbosity=4
      - -S
      - SmartSearchQueryLogVerbosity=3
      - -S
      - SmartSearchIndexLogVerbosity=3
      - -S
      - SSEVerbosity=4
      - -S
      - EventBusLogVerbosity=4
      - -S
      - EnableExtServiceLogs=1
      - -S
      - ExternalServicesLogVerbosity=4
      - -S
      - BasePathPrefix=/api
      - -S
      - FolderConnectionInterface=0
      - -S
      - Autosave=1
      - -S
      - AutosaveInterval=5
      - -S
      - ValidateJsonWebTokens=2
      - -S
      - JWKSServiceUrl=http://engine-keys:8080/v1/keys/qlik.api.internal
      - -S
      - JWTSignPrivateKeyPath=/etc/secrets/jwtPrivateKey
      - -S
      - JWTSignPrivateKeyId=uHxp0YvnYQx-AqQgHnxgKqTk_HmZiT67B8SYOlzSgoM
      - -S
      - InternalTokenServiceUrl=http://engine-edge-auth:8080/v1
      - -S
      - EnableRenewUserToken=1
      - -S
      - EnableABAC=1
      - -S
      - Gen3=1
      - -S
      - PersistenceMode=2
      - -S
      - EnableAccessControlTrace=1
      - -S
      - SystemRules=/etc/config/rules.yaml
      - -S
      - LicenseServiceUrl=http://engine-licenses:9200
      - -S
      - LicenseCacheTimeoutSeconds=3600
      - -S
      - EnableSpaces=1
      - -S
      - SpacesServiceUrl=http://engine-spaces:6080
      - -S
      - EnableEncryptData=1
      - -S
      - UseEncryptionService=1
      - -S
      - EncryptionServiceUrl=http://engine-encryption:8080
      - -S
      - EnableFeatureFlagService=1
      - -S
      - FeatureFlagServiceUrl=http://engine-feature-flags:8080
      - -S
      - EnableGroupsService=1
      - -S
      - GroupsServiceUrl=http://engine-groups:8080
      - -S
      - EnableAppExport=1
      - -S
      - EnableTempContentService=1
      - -S
      - TempContentServiceUrl=http://engine-temporary-contents:6080
      - -S
      - UseSTAN=1
      - -S
      - STANUrl=nats://engine-nats-client:4222
      - -S
      - STANCluster=engine-nats-streaming-cluster
      - -S
      - STANUseToken=1
      - -S
      - STANMaxReconnect=60
      - -S
      - STANReconnectWait=2
      - -S
      - STANTimeout=10
      - -S
      - UseEventBus=1
      - -S
      - EnvironmentName="example"
      - -S
      - RegionName="example"
      - -S
      - ShutdownWait=1
      env:
      - name: PROMETHEUS_PORT
        value: "9090"
      - name: GRPC_DNS_RESOLVER
        value: native
      image: qlik-docker-qsefe.bintray.io/engine:12.460.0
      imagePullPolicy: null
      livenessProbe:
        httpGet:
          path: /health
          port: 9076
      name: engine-reload
      ports:
      - containerPort: 9076
      - containerPort: 9090
        name: metrics
      readinessProbe:
        httpGet:
          path: /health
          port: 9076
      volumeMounts:
      - mountPath: /qlik/apps
        name: apps-storage
      - mountPath: /etc/secrets
        name: secrets-service2service
        readOnly: true
      - mountPath: /etc/config
        name: rules-volume
      - mountPath: /home/engine/Qlik/Sense
        name: storagepath
    dnsConfig:
      options:
      - name: timeout
        value: "1"
      - name: single-request-reopen
    imagePullSecrets:
    - name: artifactory-docker-secret
    terminationGracePeriodSeconds: 30
    volumes:
    - name: apps-storage
      persistentVolumeClaim:
        claimName: engine
    - name: secrets-service2service
      secret:
        secretName: engine-service-jwt-secret
    - configMap:
        defaultMode: 493
        name: engine-prestop-hook
        optional: true
      name: engine-prestop-hook
    - configMap:
        name: engine-engine-rules-cm
      name: rules-volume
    - emptyDir: {}
      name: storagepath
  workloadType: reload
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  labels:
    app: engine
  name: engine
spec:
  accessModes:
  - ReadWriteMany
  resources:
    requests:
      storage: 5Gi
`
	testCases := []struct {
		name                    string
		pluginConfig            string
		pluginInputResources    string
		expectingTransformError bool
		checkAssertions         func(*testing.T, resmap.ResMap)
	}{
				{
					name: "value_at_path_is_map",
					pluginConfig: `
apiVersion: qlik.com/v1
kind: SedOnPath
metadata:
  name: notImportantHere
path: spec/template
regex:
- s/.*//g
`,
					pluginInputResources:    basicPluginInputResources,
					expectingTransformError: true,
					checkAssertions: func(t *testing.T, resMap resmap.ResMap) {
						assert.FailNow(t, "should not be here!")
					},
				},
				{
					name: "value_at_path_is_string",
					pluginConfig: `
apiVersion: qlik.com/v1
kind: SedOnPath
metadata:
  name: notImportantHere
path: spec/template/spec/containers/name
regex:
- s/the-container/the-awesome-container/g
`,
					pluginInputResources:    basicPluginInputResources,
					expectingTransformError: false,
					checkAssertions: func(t *testing.T, resMap resmap.ResMap) {
						res := resMap.GetByIndex(0)
						assert.NotNil(t, res)

						containers, err := res.GetFieldValue("spec.template.spec.containers")
						assert.NoError(t, err)

						for _, container := range containers.([]interface{}) {
							assert.True(t, strings.HasPrefix(container.(map[string]interface{})["name"].(string), "the-awesome-container-"))
						}
					},
				},
				{
					name: "value_at_path_is_array_of_strings",
					pluginConfig: `
apiVersion: qlik.com/v1
kind: SedOnPath
metadata:
  name: notImportantHere
path: spec/template/spec/containers/command
regex:
- s/--port=.*$/--port=1234/g
- s/--TempContentServiceUrl=.*$/--TempContentServiceUrl=http:\/\/\$\(PREFIX\)-contents:6080/g
`,
					pluginInputResources:    basicPluginInputResources,
					expectingTransformError: false,
					checkAssertions: func(t *testing.T, resMap resmap.ResMap) {
						res := resMap.GetByIndex(0)
						assert.NotNil(t, res)

						containers, err := res.GetFieldValue("spec.template.spec.containers")
						assert.NoError(t, err)

						portArgSubCounter := 0
						for _, container := range containers.([]interface{}) {
							args := container.(map[string]interface{})["command"].([]interface{})
							for _, arg := range args {
								zArg := arg.(string)
								if zArg == "--port=1234" {
									portArgSubCounter++
								}
							}
						}
						assert.Equal(t, 2, portArgSubCounter)

						tempContentServiceUrlSubCounter := 0
						for _, container := range containers.([]interface{}) {
							args := container.(map[string]interface{})["command"].([]interface{})
							for _, arg := range args {
								zArg := arg.(string)
								if zArg == "--TempContentServiceUrl=http://$(PREFIX)-contents:6080" {
									tempContentServiceUrlSubCounter++
								}
							}
						}
						assert.Equal(t, 2, tempContentServiceUrlSubCounter)
					},
				},
		{
			name: "engine_args",
			pluginConfig: `
apiVersion: qlik.com/v1
kind: SedOnPath
metadata:
  name: notImportantHere
path: spec/template/spec/containers/args
regex: 
- s/--port=.*$/--port=1234/g
- s/JWKSServiceUrl=.*$/JWKSServiceUrl=http:\/\/\$\(PREFIX\)-keys:8080/g
`,
			pluginInputResources:    enginePluginInputResources,
			expectingTransformError: false,
			checkAssertions: func(t *testing.T, resMap resmap.ResMap) {
				res, err := resMap.GetById(resid.NewResId(gvk.Gvk{
					Group:   "extensions",
					Version: "v1beta1",
					Kind:    "Deployment",
				}, "engine-default"))
				assert.NoError(t, err)
				assert.NotNil(t, res)

				container, err := res.GetFieldValue("spec.template.spec.containers[0]")
				assert.NoError(t, err)
				assert.NotNil(t, container)

				jwksServiceUrlSubCounter := 0
				args := container.(map[string]interface{})["args"].([]interface{})
				for _, arg := range args {
					zArg := arg.(string)
					if zArg == "JWKSServiceUrl=http://$(PREFIX)-keys:8080" {
						jwksServiceUrlSubCounter++
					}
				}
				assert.Equal(t, 1, jwksServiceUrlSubCounter)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resourceFactory := resmap.NewFactory(resource.NewFactory(
				kunstruct.NewKunstructuredFactoryImpl()), transformer.NewFactoryImpl())

			resMap, err := resourceFactory.NewResMapFromBytes([]byte(testCase.pluginInputResources))
			if err != nil {
				t.Fatalf("Err: %v", err)
			}

			err = KustomizePlugin.Config(loadertest.NewFakeLoader("/"), resourceFactory, []byte(testCase.pluginConfig))
			if err != nil {
				t.Fatalf("Err: %v", err)
			}

			err = KustomizePlugin.Transform(resMap)
			if err != nil && testCase.expectingTransformError {
				return
			}

			if err != nil {
				t.Fatalf("Err: %v", err)
			}

			for _, res := range resMap.Resources() {
				fmt.Printf("--res: %v\n", res.String())
			}

			testCase.checkAssertions(t, resMap)
		})
	}
}
