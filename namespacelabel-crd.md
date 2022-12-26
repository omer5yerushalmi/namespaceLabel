# API Reference

## Packages
- [omer.omer.io/v1](#omeromeriov1)


## omer.omer.io/v1

Package v1 contains API Schema definitions for the omer v1 API group

### Resource Types
- [NamespaceLabel](#namespacelabel)
- [NamespaceLabelList](#namespacelabellist)



#### NamespaceLabel



NamespaceLabel is the Schema for the namespacelabels API

_Appears in:_
- [NamespaceLabelList](#namespacelabellist)

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `omer.omer.io/v1`
| `kind` _string_ | `NamespaceLabel`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[NamespaceLabelSpec](#namespacelabelspec)_ |  |


#### NamespaceLabelList



NamespaceLabelList contains a list of NamespaceLabel



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `omer.omer.io/v1`
| `kind` _string_ | `NamespaceLabelList`
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[NamespaceLabel](#namespacelabel) array_ |  |


#### NamespaceLabelSpec



NamespaceLabelSpec defines the desired state of NamespaceLabel

_Appears in:_
- [NamespaceLabel](#namespacelabel)

| Field | Description |
| --- | --- |
| `labels` _object (keys:string, values:string)_ |  |




