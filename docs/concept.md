# Project: Bulward

Multi tenancy - multi user management for Kubernetes.

## Motivation

Kubermatic and KubeCarrier both need a system to manage users, projects and RBAC. In Kubermatic this functionallity is currently handled

## Issues

### Owners cannot kill their own permissions

Owner permissions are reconciled, if deleted or altered. A validating webhook will prevent the last owner from beeing removed.

### Users can only see organizations and projects they are assigned to

`Organization` and `Project` objects are exposed via a kubernetes extension API server. This extension API server will use the authenticated username to filter the list of `Organizations` and `Projects` to only contain items that the user is part of.

A user is part of a project if:
- he is one of the Owners
- he is referenced in a `RoleBinding` within the `Organization` or `Project`, beeing referenced in the `Project` will also allow the user to see the `Organization`.

### Extension API needs it's own etcd

Yes, but actually no. There are examples, where we can use CRDs (or annotations of k8s objects) as backend storage for our own API, so we don't require another etcd as storage.

Thats why we have `Organization` and `InternalOrganization`, `Organization` is meant to be exposed to users via the apiserver, while `InternalOrganization` objects are just for storing data.

### Organization names can be set by enduser

As Organizations are kind-a Cluster-Scoped, users cannot set their own object name, or they can run into name conflicts with objects they cannot even see.

To solve this, the cluster scoped `InternalOrganization` object is always using `generateName` and gets a `bulward.io/name` label with the original object name.
The extension apiserver will expose `Organization` objects with the original name, making this intransparent to the APIs user.

### Organization/Project Owners can manage their own Roles

Organization and project owners are automatically granted permission to create new `Role` and `RoleBinding` objects. The Kubernetes API Server ensures safty against priviledge escalation.

Other users are managed via `RoleBindings`.

### Extenable Default Roles for Organizations and Projects

Tools like Kubermatic and KubeCarrier will bring their own default roles and permissions, or are even managing dynamic Roles depending on available CRDs (KubeCarrier).
Bulward should make it easy to integrate these Roles without needing deep integrations.

> See #Predefined Roles section

### Preserve Audit-Logging capability

Users and Service Accounts are interacting with the kube-apiserver using their username/tokens, which can be used in concert with Kubernetes build-in Audit-Logging.

## APIs

### Organization

`Organization` objects are the top-level organisational objects in the multi-user hirachy that Bulward provides. Permissions to manage `Organizations` can be granted via normal `ClusterRoles` and `ClusterRoleBindings`.

Every Organization will have a Kubernetes Namespace assigned, which is used to interact with objects belonging to this Organization.

User Facing API (via extension apiserver):

```yaml
apiVersion: bulward.io/v1alpha1
kind: Organization
metadata:
  name: loodse
spec:
  owners:
  - kind: User
    name: hannibal@company.com
    apiGroup: rbac.authorization.k8s.io
status:
  namespace:
    name: loodse-xxx
  members:
  # list of all members 
  # (owners + subjects referenced in RoleBindings in Organization and Project Namespaces)
  - kind: User
    name: hannibal@company.com
    apiGroup: rbac.authorization.k8s.io
  - kind: ServiceAccount
    name: robot-3000
    apiGroup: v1
  - kind: User
    name: hans@company.com
    apiGroup: rbac.authorization.k8s.io
  conditions: []
  observedGeneration: 0
```

ClusterScoped `CustomResourceDefinition`, used as backend storage for the user facing `Organization` object.

```yaml
apiVersion: bulward.io/v1alpha1
kind: InternalOrganization
metadata:
  name: loodse-xxx
  generateName: loodse
  label:
    bulward.io/name: loodse
spec:
  owners:
  - kind: User
    name: hannibal@company.com
    apiGroup: rbac.authorization.k8s.io
status:
  namespace:
    name: loodse-xxx
  members:
  # list of all members 
  # (owners + subjects referenced in RoleBindings in Organization and Project Namespaces)
  - kind: User
    name: hannibal@company.com
    apiGroup: rbac.authorization.k8s.io
  - kind: ServiceAccount
    name: robot-3000
    apiGroup: v1
  - kind: User
    name: hans@company.com
    apiGroup: rbac.authorization.k8s.io
  conditions: []
  observedGeneration: 0
```

### Project

`Project` objects allow `Organizations` to sub-organize "stuff".
Every Project will have a Kubernetes Namespace assigned, which is used to interact with objects belonging to this Project.

User Facing API (via extension apiserver):

```yaml
apiVersion: bulward.io/v1alpha1
kind: Project
metadata:
  name: project-01
  # lives in Org-Namespace
  namespace: loodse-xxx
spec:
  owners:
  - kind: User
    name: hannibal@company.com
    apiGroup: rbac.authorization.k8s.io
status:
  namespace:
    name: project-01-xxx
  members:
  # list of all members 
  # (owners + subjects referenced in RoleBindings in the Project Namespace)
  - kind: User
    name: hannibal@company.com
    apiGroup: rbac.authorization.k8s.io
  - kind: ServiceAccount
    name: robot-3000
    apiGroup: v1
  - kind: User
    name: hans@company.com
    apiGroup: rbac.authorization.k8s.io
  conditions: []
  observedGeneration: 0
```

NamespaceScoped `CustomResourceDefinition`, used as backend storage for the user facing `Project` object.

```yaml
apiVersion: bulward.io/v1alpha1
kind: InternalProject
metadata:
  name: project-01
  # lives in Org-Namespace
  namespace: loodse-xxx
spec:
  owners:
  - kind: User
    name: hannibal@company.com
    apiGroup: rbac.authorization.k8s.io
status:
  namespace:
    name: project-01-xxx
  members:
  # list of all members 
  # (owners + subjects referenced in RoleBindings in the Project Namespace)
  - kind: User
    name: hannibal@company.com
    apiGroup: rbac.authorization.k8s.io
  - kind: ServiceAccount
    name: robot-3000
    apiGroup: v1
  - kind: User
    name: hans@company.com
    apiGroup: rbac.authorization.k8s.io
  conditions: []
  observedGeneration: 0
```

### OrganizationRoleTemplate

`OrganizationRoleTemplate` objects are reconciled into `Role` objects into every Organization or Project namespace.
If specified via the `bindTo` parameter, a `RoleBinding` for Owners of the `Organization` is also created and reconciled.

Default minimal `OrganizationRoleTemplates` are listed below. Addiotional default roles can be added by each integration (Kubermatic/KubeCarrier) or vendor.

```yaml
# Project Admins can manage projects within an Organization.
apiVersion: bulward.io/v1alpha1
kind: OrganizationRoleTemplate
metadata:
  name: project-admin
spec:
  scopes:
  - Organization # will be created as Role in every Organization Namespace
  bindTo:
  - Owners
  rules:
  - apiGroups:
    - bulward.io
    resources:
    - projects
    verbs:
    - get
    - list
    - watch
    - create
    - update
    - patch
    - delete
status:
  conditions: []
  targets:
  # tracking rollout of the ... Role to different targets
  - kind: Organization
    name: loodse
    apiGroup: bulward.io
    observedGeneration: 0
---
# RBAC Admins can create new Roles and RoleBindings.
# RoleBindings are checked by k8s, so priviledge escalation is not possible.
apiVersion: bulward.io/v1alpha1
kind: OrganizationRoleTemplate
metadata:
  name: rbac-admin
spec:
  scopes:
  - Organization # will be created as Role in every Organization Namespace
  - Project # will be created as Role in every Project Namespace
  bindTo:
  - Owners
  rules:
  - apiGroups:
    - rbac.authorization.k8s.io
    resources:
    - roles
    - rolebindings
    verbs:
    - get
    - list
    - watch
    - create
    - update
    - patch
    - delete
    - bind
    # - escalate <- never ever grant this
status:
  conditions: []
  targets:
  # tracking rollout of the ... Role to different targets
  - kind: Organization
    name: loodse
    apiGroup: bulward.io
    observedGeneration: 0
  - kind: Project
    name: project-01
    apiGroup: bulward.io
    observedGeneration: 0
```

## ProjectRoleTemplate

`ProjectRoleTemplate` could be used by Organization Owners to manage the same `Role` across multiple `Projects`.

```yaml
apiVersion: bulward.io/v1alpha1
kind: ProjectRoleTemplate
metadata:
  name: rbac-admin
spec:
  bindTo:
  - Everyone # (maybe?) every member already mentioned in a RoleBinding
  labelSelector: {} # (maybe?) limit these roles to only a few of your projects
  rules:
  - apiGroups:
    - my-corp.com
    resources:
    - mycoolapps
    verbs:
    - get
    - list
    - watch
    - create
    - update
    - patch
    - delete
status:
  conditions: []
  targets:
  # tracking rollout of the ... Role to different targets
  - kind: Project
    name: project-01
    apiGroup: bulward.io
    observedGeneration: 0
```

## Open Issues TBD

### Cleanup/Revoke permissions removed from Template

Users can create their own `Roles` and bind them to users `RoleBinding` as long as they don't exceed their own permission level (escalate is not allowed).

Although if permissions are removed from the `OrganizationRoleTemplates` or `ProjectRoleTemplate` custom Roles are unaltered (as they are not tracked by the system), so Users may retain access.

### Possible Solution

A solution to solve this, could be a custom controller altering user created roles (and removing rules that are revoked in the template via an intersect). This would also prevent users from creating `Roles` with rules that they are not allowed to bind to.
