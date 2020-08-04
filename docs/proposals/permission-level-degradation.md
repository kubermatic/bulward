# Bulward Permission Level Degradation

This document extends on the initial Bulward concept to enable degradation of user defined roles.

## Background

After installing bulward organization owners will have very limited permissions. Essentially they will only be allowed to interact with the Bulward API -> e.g. creating new Projects, organize users and create sub-roles.

In order to actually grant an organization rights to do something, these rights need to be explicitly assigned to an Organization.
New Operators may ship `OrganizationRoleTemplates` to setup default roles to interact with it.

This is all fine, but it gets complicated when Organization Owners define their own sub-roles.

While granting new permissions is uncomplicated, removing permissions is less easy, because new roles utilizing these permissions might have been created.
This concept deals with the necessary permission/role cleanup, by introducing new theoretical concepts and extending the KubeCarrier API.

## Concepts

### Max Permission Level (Organization/Project)

Organizations and Projects have both a maximum permission level that cannot be exceeded within.
This maximum permission level results from all roles that are assigned to the organization unit and thus to the Organization/Project Owner.

Organization/Project Owners may create sub-roles that don't exceed this maximum permission level.

If roles are removed from the organization the maximum permission level drops and sub-roles need to be limited in order not to exceed this permission level.

### Don't edit user defined roles

When users define their own `Roles` within their Project or Organization, bulward should not update those `Roles` to "fix" roles that violate the max permission level.
Changing user defined specification is very confusing to administrators and hard to make transparent to users.

Instead we should define our own Role/RoleBinding objects for each permission scope (Project/Organization) and translate that into Kubernetes `Role`/`RoleBinding` objects.
This way we can keep the user specification intact, while enabling only parts of his specification when it violates the max permission level.
We can also report detailed 

### New CRDs

## ProjectRole

`ProjectRole` objects control a `Role` object within the `Project` namespace, that the object was created in.
In difference to Kubernetes `Role` objects, Bulward will only consider RBAC rules in the definition that are not in violation of the max permission level of the project.

The accepted rules will be listed in the objects status to inform the user.

`ProjectRoles` are validated on creation and update to enforce the same prerequisites as Kubernetes:
https://kubernetes.io/docs/reference/access-authn-authz/rbac/#restrictions-on-role-creation-or-update

```yaml
apiVersion: bulward.io/v1alpha1
kind: ProjectRole
metadata:
  name: mycoolapps-admin
  namespace: project-namespace
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
  phase: Established
  observedGeneration: 1
  conditions: []
  acceptedRules:
  - apiGroups:
    - my-corp.com
    resources:
    - mycoolapps
    verbs: # limited verbs, due to a max permission issue
    - get
    - list
    - watch
    - update
    - patch
```

## OrganizationRole

`OrganizationRole` objects control a `Role` object within the `Organization` namespace and all `Project` namespaces existing within the organization.
In difference to Kubernetes `Role` objects, Bulward will only consider RBAC rules in the definition that are not in violation of the max permission level of the organization.

The accepted rules will be listed in the objects status to inform the user.

`OrganizationRoles` are validated on creation and update to enforce the same prerequisites as Kubernetes:
https://kubernetes.io/docs/reference/access-authn-authz/rbac/#restrictions-on-role-creation-or-update

```yaml
apiVersion: bulward.io/v1alpha1
kind: OrganizationRole
metadata:
  name: mycoolapps-admin
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
  phase: Established
  observedGeneration: 1
  conditions: []
  acceptedRules:
  - apiGroups:
    - my-corp.com
    resources:
    - mycoolapps
    verbs:
    - get
    - list
    - watch
    - update
    - patch
```

## ProjectRoleBinding

`ProjectRoleBinding` binds a `ProjectRole` or a `OrganizationRole` to a list of RBAC subjects.
The only valid `roleRef` types are `ProjectRole` or `OrganizationRole` objects from the `bulward.io` api.

`ProjectRoleBinding` objects are validated on creation and update to enforce the same prerequisites as Kubernetes:
https://kubernetes.io/docs/reference/access-authn-authz/rbac/#restrictions-on-role-binding-creation-or-update

```yaml
apiVersion: bulward.io/v1alpha1
kind: ProjectRoleBinding
metadata:
  name: mycoolapps-admin
roleRef:
  kind: ProjectRole # or OrganizationRole
  apiGroup: bulward.io
  name: mycoolapps-admin
subjects:
- kind: User
  apiGroup: rbac.authorization.k8s.io
  name: sebastian@kubermatic.com
status:
  phase: Established
  observedGeneration: 1
  conditions: []
```

## OrganizationRoleBinding

`OrganizationRoleBinding` binds a `OrganizationRole` to a list of RBAC subjects across all projects and the organization namespace.
The only valid `roleRef` type is the `OrganizationRole` from the `bulward.io` api.

`OrganizationRoleBinding` objects are validated on creation and update to enforce the same prerequisites as Kubernetes:
https://kubernetes.io/docs/reference/access-authn-authz/rbac/#restrictions-on-role-binding-creation-or-update

```yaml
apiVersion: bulward.io/v1alpha1
kind: OrganizationRoleBinding
metadata:
  name: mycoolapps-admin
roleRef:
  kind: OrganizationRole
  apiGroup: bulward.io
  name: mycoolapps-admin
subjects:
- kind: User
  apiGroup: rbac.authorization.k8s.io
  name: sebastian@kubermatic.com
status:
  phase: Established
  observedGeneration: 1
  conditions: []
```

## Changes from initial concept

### ProjectRoleTemplate and OrganizationRoleTemplate

`ProjectRoleTemplate` and `OrganizationRoleTemplate` objects should only create bulward `*Role`/`*RoleBinding` objects, instead of the Kubernetes-native types.
