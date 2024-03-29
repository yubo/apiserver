/*
Copyright 2022 The Kubernetes Authors.

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

package features

import (
	"github.com/yubo/apiserver/components/featuregate"
)

const (
	// owner: @logicalhan
	// kep: https://kep.k8s.io/3466
	// alpha: v1.26
	ComponentSLIs featuregate.Feature = "ComponentSLIs"
)

func featureGates() map[featuregate.Feature]featuregate.FeatureSpec {
	return map[featuregate.Feature]featuregate.FeatureSpec{
		ComponentSLIs: {Default: false, PreRelease: featuregate.Alpha},
	}
}

// AddFeatureGates adds all feature gates used by this package.
func AddFeatureGates(mutableFeatureGate featuregate.MutableFeatureGate) error {
	return mutableFeatureGate.Add(featureGates())
}
