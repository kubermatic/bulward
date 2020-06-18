# Project: Bulward

Multi tenancy - multi user management for Kubernetes.

## Introduction

Both Kubermatic and KubeCarrier need a system to manage multiple independent users, their projects and RBAC permissions.

Kubermatic is currently handling this functionality with custom code in the kubermatic apiserver. While the same pattern could be used for KubeCarrier, the code cannot be directly reused.

KubeCarrier is currently offloading all RBAC and user management into Kubernetes by managing multiple Namespaces and creating RBAC Roles and RoleBindings. Although to support more advanced use cases, this system needs to be heavily extended.

## Requirements

1. Organization level to group multiple projects for common access control
2. Project level to group "things" (Kubermatic User Clusters or KubeCarrier Service Instances)
3. Users can manage their own Projects and Organizations (Create, Update, Delete...)
4. Audit Logging (Kubernetes Knows the real user)
5. Users can manage custom Roles within their Organizations/Projects
6. Users can not orphan a project or Organization (leave a project/organization without owners)

## Features

### Users can manage their own Projects and Organizations

`Organization` and `Project` objects are exposed via a kubernetes extension API server. This extension API server will use the authenticated username to filter the list of `Organizations` and `Projects` to only contain items that the user is part of.

Users is part of a project if:
- they are one of the Owners
- they are referenced in a `RoleBinding` within the `Organization` or `Project`, being referenced in the `Project` will also allow the user to see the `Organization`.

### Users can manage custom Roles within their Organizations/Projects

Organization and project owners are automatically granted permission to create new `Role` and `RoleBinding` objects. The Kubernetes API Server ensures safety against privilege escalation.

Other users are managed via `RoleBindings`.

### Users can not orphan a project or Organization

Owner permissions are reconciled, if deleted or altered. A validating webhook will prevent the last owner of an Organization or Project from being removed.

### Organization names can be set by end user

As Organizations are kind-a Cluster-Scoped, users cannot set their own object name, or they can run into name conflicts with objects they cannot even see.

To solve this, the cluster scoped `InternalOrganization` object is always using `generateName` and gets a `bulward.io/name` label with the original object name.
The extension apiserver will expose `Organization` objects with the original name, making this nontransparent to the APIs user.

### Extensible Default Roles for Organizations and Projects

Tools like Kubermatic and KubeCarrier will bring their own default roles and permissions, or are even managing dynamic Roles depending on available CRDs (KubeCarrier).
Bulward should make it easy to integrate these Roles without needing deep integrations.

> See #Predefined Roles section

### Preserve Audit-Logging capability

Users and Service Accounts are interacting with the kube-apiserver using their username/tokens, which can be used in concert with Kubernetes build-in Audit-Logging.

### Extension API needs it's own etcd

Yes, but actually no. There are examples, where we can use CRDs (or annotations of k8s objects) as backend storage for our own API, so we don't require another etcd as storage.

Thats why we have `Organization` and `InternalOrganization`, `Organization` is meant to be exposed to users via the apiserver, while `InternalOrganization` objects are just for storing data.

## APIs

### Organization

`Organization` objects are the top-level organizational objects in the multi-user hierarchy that Bulward provides. Permissions to manage `Organizations` can be granted via normal `ClusterRoles` and `ClusterRoleBindings`.

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
# RoleBindings are checked by k8s, so privilege escalation is not possible.
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

### Orchestrating Projects across clusters

Kubermatic is working across multiple Seed Clusters and the same namespace isolation needs to be setup in these Seed Clusters in order for us to fully utilize this project in Kubermatic.

### Cleanup/Revoke permissions removed from Template

Users can create their own `Roles` and bind them to users `RoleBinding` as long as they don't exceed their own permission level (escalate is not allowed).

Although if permissions are removed from the `OrganizationRoleTemplates` or `ProjectRoleTemplate` custom Roles are unaltered (as they are not tracked by the system), so Users may retain access.

**Possible Solution**

A solution to solve this, could be a custom controller altering user created roles (and removing rules that are revoked in the template via an intersect). This would also prevent users from creating `Roles` with rules that they are not allowed to bind to.

## Feature Considerations

### Setup Network Isolation - NetworkPolicies

### Setup Quotas - ResourceQuotas

### Cluster Resource Access

Nodes and Persistent Volumes are Cluster-Level resources and for a general purpose multi tenancy solution, we probably want to look into this topic.
e.g. Open Shift let's you assign Nodes to Projects via Selectors, so tenants can have dedicated nodes.

## Other Projects

Projects that are active in this field or solve similar issues.

### Kubernetes Virtual Cluster - Alpha/Experimental

VirtualCluster represents a new architecture to address various Kubernetes control plane isolation challenges. It extends existing namespace based Kubernetes multi-tenancy model by providing each tenant a cluster view. VirtualCluster completely leverages Kubernetes extendability and preserves full API compatibility. That being said, the core Kubernetes components are not modified in virtual cluster. [...]

https://github.com/kubernetes-sigs/multi-tenancy/tree/master/incubator/virtualcluster

### Kubernetes Hierarchical Namespace Controller

Hierarchical namespaces make it easier for you to create and manage namespaces in your cluster. For example, you can create a hierarchical namespace under your team's namespace, even if you don't have cluster-level permission to create namespaces, and easily apply policies like RBAC and Network Policies across all namespaces in your team (e.g. a set of related microservices). [...]

https://github.com/kubernetes-sigs/multi-tenancy/tree/master/incubator/hnc
Concept: https://docs.google.com/document/d/10MZfFfbQMm33CBboMq2bfrEtXkJQQT4-UH4DDXZRrKY

### Kiosk - Alpha

The core idea of kiosk is to use Kubernetes namespaces as isolated workspaces where tenant applications can run isolated from each other. To minimize admin overhead, cluster admins are supposed to configure kiosk which then becomes a self-service system for provisioning Kubernetes namespaces for tenants. [...]

https://github.com/kiosk-sh/kiosk

### OpenShift

OpenShift implements some multi tenancy mechanisms and controllers to make large enterprise installations viable.
