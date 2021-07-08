// Copyright 2021 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package singleton

import (
	"context"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ manager.Runnable = runnable{}
var _ manager.LeaderElectionRunnable = runnable{}

type runnable struct {
	client.Client

	objs []client.Object
}

// NewRunnable returns a manager.Runnable that requires leader election to
// create all objs using c. This runnable should be added to a manager.Manager
// with Manager.Add(runnable).
func NewRunnable(c client.Client, objs ...client.Object) manager.Runnable {
	return runnable{Client: c, objs: objs}
}

func (r runnable) NeedLeaderElection() bool { return true }

func (r runnable) Start(ctx context.Context) error {
	switch len(r.objs) {
	case 0:
		return nil
	case 1:
		return r.create(ctx, r.objs[0])
	}

	fs := make([]func() error, len(r.objs))
	for i := range r.objs {
		fs[i] = func() error { return r.create(ctx, r.objs[i]) }
	}
	return utilerrors.AggregateGoroutines(fs...)
}

func (r runnable) create(ctx context.Context, obj client.Object) error {
	if err := r.Create(ctx, obj); err != nil {
		return err
	}

	key := types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}
	return wait.PollImmediateUntil(200*time.Millisecond, func() (bool, error) {
		err := r.Get(ctx, key, obj)
		if err != nil && apierrors.IsNotFound(err) {
			return false, nil
		}
		return err == nil, err
	}, ctx.Done())
}
