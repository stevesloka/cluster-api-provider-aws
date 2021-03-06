// Copyright © 2018 The Kubernetes Authors.
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

package ec2

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"
	"k8s.io/klog"
	"sigs.k8s.io/cluster-api-provider-aws/pkg/apis/awsprovider/v1alpha1"
	"sigs.k8s.io/cluster-api-provider-aws/pkg/cloud/aws/filter"
	"sigs.k8s.io/cluster-api-provider-aws/pkg/cloud/aws/services/awserrors"
	"sigs.k8s.io/cluster-api-provider-aws/pkg/cloud/aws/tags"
)

const (
	defaultVpcCidr = "10.0.0.0/16"
)

func (s *Service) reconcileVPC() error {
	klog.V(2).Infof("Reconciling VPC")

	vpc, err := s.describeVPC()
	if awserrors.IsNotFound(err) {
		// Create a new vpc.
		vpc, err = s.createVPC()
		if err != nil {
			return errors.Wrap(err, "failed to create new vpc")
		}

	} else if err != nil {
		return errors.Wrap(err, "failed to describe VPCs")
	}

	vpc.DeepCopyInto(s.scope.VPC())
	klog.V(2).Infof("Working on VPC %q", vpc.ID)
	return nil
}

func (s *Service) createVPC() (*v1alpha1.VPC, error) {
	if s.scope.VPC().CidrBlock == "" {
		s.scope.VPC().CidrBlock = defaultVpcCidr
	}

	input := &ec2.CreateVpcInput{
		CidrBlock: aws.String(s.scope.VPC().CidrBlock),
	}

	out, err := s.scope.EC2.CreateVpc(input)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create vpc")
	}

	wReq := &ec2.DescribeVpcsInput{VpcIds: []*string{out.Vpc.VpcId}}
	if err := s.scope.EC2.WaitUntilVpcAvailable(wReq); err != nil {
		return nil, errors.Wrapf(err, "failed to wait for vpc %q", *out.Vpc.VpcId)
	}

	name := fmt.Sprintf("%s-vpc", s.scope.Name())

	applyTagsParams := &tags.ApplyParams{
		EC2Client: s.scope.EC2,
		BuildParams: tags.BuildParams{
			ClusterName: s.scope.Name(),
			ResourceID:  *out.Vpc.VpcId,
			Lifecycle:   tags.ResourceLifecycleOwned,
			Name:        aws.String(name),
			Role:        aws.String(tags.ValueCommonRole),
		},
	}

	if err := tags.Apply(applyTagsParams); err != nil {
		return nil, errors.Wrapf(err, "failed to tag vpc %q", *out.Vpc.VpcId)
	}

	klog.V(2).Infof("Created new VPC %q with cidr %q", *out.Vpc.VpcId, *out.Vpc.CidrBlock)

	return &v1alpha1.VPC{
		ID:        *out.Vpc.VpcId,
		CidrBlock: *out.Vpc.CidrBlock,
	}, nil
}

func (s *Service) deleteVPC() error {
	// TODO(johanneswuerbach): ensure that the VPC is owned by this cluster before deleting
	input := &ec2.DeleteVpcInput{
		VpcId: aws.String(s.scope.VPC().ID),
	}

	_, err := s.scope.EC2.DeleteVpc(input)
	if err != nil {
		// Ignore if it's already deleted
		if code, ok := awserrors.Code(err); code != "InvalidVpcID.NotFound" && ok {
			return errors.Wrapf(err, "failed to delete vpc %q", s.scope.VPC().ID)
		}
		return err
	}

	klog.V(2).Infof("Deleted VPC %q", s.scope.VPC().ID)
	return nil
}

func (s *Service) describeVPC() (*v1alpha1.VPC, error) {
	input := &ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			filter.EC2.VPCStates(ec2.VpcStatePending, ec2.VpcStateAvailable),
		},
	}

	if s.scope.VPC().ID == "" {
		// Try to find a previously created and tagged VPC
		input.Filters = []*ec2.Filter{filter.EC2.Cluster(s.scope.Name())}
	} else {
		input.VpcIds = []*string{aws.String(s.scope.VPC().ID)}
	}

	out, err := s.scope.EC2.DescribeVpcs(input)
	if err != nil {
		if awserrors.IsNotFound(err) {
			return nil, err
		}

		return nil, errors.Wrap(err, "failed to query ec2 for VPCs")
	}

	if len(out.Vpcs) == 0 {
		return nil, awserrors.NewNotFound(errors.Errorf("could not find vpc %q", s.scope.VPC().ID))
	} else if len(out.Vpcs) > 1 {
		return nil, awserrors.NewConflict(errors.Errorf("found more than one vpc with supplied filters. Please clean up extra VPCs: %s", out.GoString()))
	}

	switch *out.Vpcs[0].State {
	case ec2.VpcStateAvailable, ec2.VpcStatePending:
	default:
		return nil, awserrors.NewNotFound(errors.Errorf("could not find available or pending vpc"))
	}

	return &v1alpha1.VPC{
		ID:        *out.Vpcs[0].VpcId,
		CidrBlock: *out.Vpcs[0].CidrBlock,
	}, nil
}
