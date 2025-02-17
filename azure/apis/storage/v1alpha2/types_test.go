/*
Copyright 2019 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha2

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2017-06-01/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/google/go-cmp/cmp"
	"github.com/onsi/gomega"
	"golang.org/x/net/context"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	runtimev1alpha1 "github.com/crossplaneio/crossplane-runtime/apis/core/v1alpha1"
	"github.com/crossplaneio/crossplane-runtime/pkg/resource"
	"github.com/crossplaneio/crossplane-runtime/pkg/test"

	localtest "github.com/crossplaneio/stack-azure/pkg/test"
)

const (
	namespace = "default"
	name      = "test-instance"
)

var (
	c client.Client
)

var (
	_ resource.Managed = &Account{}
	_ resource.Managed = &Container{}
)

func TestMain(m *testing.M) {
	t := test.NewEnv(namespace, SchemeBuilder.SchemeBuilder, localtest.CRDs())
	c = t.StartClient()
	t.StopAndExit(m.Run())
}

func TestAzureStorageAccount(t *testing.T) {
	key := types.NamespacedName{Name: name, Namespace: namespace}
	created := &Account{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: AccountSpec{
			AccountParameters: AccountParameters{
				ResourceGroupName:  "test-group",
				StorageAccountName: "test-name",
				StorageAccountSpec: &StorageAccountSpec{
					Sku: &Sku{
						Name: storage.StandardLRS,
					},
					Kind: storage.Storage,
				},
			},
			ResourceSpec: runtimev1alpha1.ResourceSpec{
				ProviderReference: &core.ObjectReference{},
			},
		},
	}
	g := gomega.NewGomegaWithT(t)

	// Test Create
	fetched := &Account{}
	g.Expect(c.Create(context.TODO(), created)).NotTo(gomega.HaveOccurred())

	g.Expect(c.Get(context.TODO(), key, fetched)).NotTo(gomega.HaveOccurred())
	g.Expect(fetched).To(gomega.Equal(created))

	// Test Updating the Labels
	updated := fetched.DeepCopy()
	updated.Labels = map[string]string{"hello": "world"}
	g.Expect(c.Update(context.TODO(), updated)).NotTo(gomega.HaveOccurred())

	g.Expect(c.Get(context.TODO(), key, fetched)).NotTo(gomega.HaveOccurred())
	g.Expect(fetched).To(gomega.Equal(updated))

	// Test Delete
	g.Expect(c.Delete(context.TODO(), fetched)).NotTo(gomega.HaveOccurred())
	g.Expect(c.Get(context.TODO(), key, fetched)).To(gomega.HaveOccurred())
}

func TestContainer_GetContainerName(t *testing.T) {
	om := metav1.ObjectMeta{
		Namespace: "foo",
		Name:      "bar",
		UID:       "test-uid",
	}
	type fields struct {
		ObjectMeta metav1.ObjectMeta
		Spec       ContainerSpec
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "no name format",
			fields: fields{
				ObjectMeta: om,
				Spec:       ContainerSpec{},
			},
			want: "test-uid",
		},
		{
			name: "format string",
			fields: fields{
				ObjectMeta: om,
				Spec: ContainerSpec{
					ContainerParameters: ContainerParameters{
						NameFormat: "foo-%s",
					},
				},
			},
			want: "foo-test-uid",
		},
		{
			name: "constant string",
			fields: fields{
				ObjectMeta: om,
				Spec: ContainerSpec{
					ContainerParameters: ContainerParameters{
						NameFormat: "foo-bar",
					},
				},
			},
			want: "foo-bar",
		},
		{
			name: "invalid: multiple substitutions",
			fields: fields{
				ObjectMeta: om,
				Spec: ContainerSpec{
					ContainerParameters: ContainerParameters{
						NameFormat: "foo-%s-bar-%s",
					},
				},
			},
			want: "foo-test-uid-bar-%!s(MISSING)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Container{
				ObjectMeta: tt.fields.ObjectMeta,
				Spec:       tt.fields.Spec,
			}
			if got := c.GetContainerName(); got != tt.want {
				t.Errorf("Container.GetContainerName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parsePublicAccessType(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want azblob.PublicAccessType
	}{
		{name: "none", args: args{s: ""}, want: azblob.PublicAccessNone},
		{name: "blob", args: args{s: "blob"}, want: azblob.PublicAccessBlob},
		{name: "container", args: args{s: "container"}, want: azblob.PublicAccessContainer},
		{name: "other", args: args{s: "other"}, want: "other"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePublicAccessType(tt.args.s)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("parsePublicAccessType(): got != want:\n%s", diff)
			}
		})
	}
}
