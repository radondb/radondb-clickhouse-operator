/*
Copyright The Kubernetes Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	clickhouseradondbcomv1 "github.com/radondb/clickhouse-operator/pkg/apis/clickhouse.radondb.com/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeClickHouseBackups implements ClickHouseBackupInterface
type FakeClickHouseBackups struct {
	Fake *FakeClickhouseV1
	ns   string
}

var clickhousebackupsResource = schema.GroupVersionResource{Group: "clickhouse.radondb.com", Version: "v1", Resource: "clickhousebackups"}

var clickhousebackupsKind = schema.GroupVersionKind{Group: "clickhouse.radondb.com", Version: "v1", Kind: "ClickHouseBackup"}

// Get takes name of the clickHouseBackup, and returns the corresponding clickHouseBackup object, and an error if there is any.
func (c *FakeClickHouseBackups) Get(ctx context.Context, name string, options v1.GetOptions) (result *clickhouseradondbcomv1.ClickHouseBackup, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(clickhousebackupsResource, c.ns, name), &clickhouseradondbcomv1.ClickHouseBackup{})

	if obj == nil {
		return nil, err
	}
	return obj.(*clickhouseradondbcomv1.ClickHouseBackup), err
}

// List takes label and field selectors, and returns the list of ClickHouseBackups that match those selectors.
func (c *FakeClickHouseBackups) List(ctx context.Context, opts v1.ListOptions) (result *clickhouseradondbcomv1.ClickHouseBackupList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(clickhousebackupsResource, clickhousebackupsKind, c.ns, opts), &clickhouseradondbcomv1.ClickHouseBackupList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &clickhouseradondbcomv1.ClickHouseBackupList{ListMeta: obj.(*clickhouseradondbcomv1.ClickHouseBackupList).ListMeta}
	for _, item := range obj.(*clickhouseradondbcomv1.ClickHouseBackupList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested clickHouseBackups.
func (c *FakeClickHouseBackups) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(clickhousebackupsResource, c.ns, opts))

}

// Create takes the representation of a clickHouseBackup and creates it.  Returns the server's representation of the clickHouseBackup, and an error, if there is any.
func (c *FakeClickHouseBackups) Create(ctx context.Context, clickHouseBackup *clickhouseradondbcomv1.ClickHouseBackup, opts v1.CreateOptions) (result *clickhouseradondbcomv1.ClickHouseBackup, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(clickhousebackupsResource, c.ns, clickHouseBackup), &clickhouseradondbcomv1.ClickHouseBackup{})

	if obj == nil {
		return nil, err
	}
	return obj.(*clickhouseradondbcomv1.ClickHouseBackup), err
}

// Update takes the representation of a clickHouseBackup and updates it. Returns the server's representation of the clickHouseBackup, and an error, if there is any.
func (c *FakeClickHouseBackups) Update(ctx context.Context, clickHouseBackup *clickhouseradondbcomv1.ClickHouseBackup, opts v1.UpdateOptions) (result *clickhouseradondbcomv1.ClickHouseBackup, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(clickhousebackupsResource, c.ns, clickHouseBackup), &clickhouseradondbcomv1.ClickHouseBackup{})

	if obj == nil {
		return nil, err
	}
	return obj.(*clickhouseradondbcomv1.ClickHouseBackup), err
}

// Delete takes name of the clickHouseBackup and deletes it. Returns an error if one occurs.
func (c *FakeClickHouseBackups) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(clickhousebackupsResource, c.ns, name), &clickhouseradondbcomv1.ClickHouseBackup{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeClickHouseBackups) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(clickhousebackupsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &clickhouseradondbcomv1.ClickHouseBackupList{})
	return err
}

// Patch applies the patch and returns the patched clickHouseBackup.
func (c *FakeClickHouseBackups) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *clickhouseradondbcomv1.ClickHouseBackup, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(clickhousebackupsResource, c.ns, name, pt, data, subresources...), &clickhouseradondbcomv1.ClickHouseBackup{})

	if obj == nil {
		return nil, err
	}
	return obj.(*clickhouseradondbcomv1.ClickHouseBackup), err
}
