package buildpackage_test

import (
	"testing"

	"github.com/buildpack/imgutil/fakes"
	"github.com/golang/mock/gomock"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpack/pack/api"
	"github.com/buildpack/pack/buildpackage"
	"github.com/buildpack/pack/dist"
	ifakes "github.com/buildpack/pack/internal/fakes"
	h "github.com/buildpack/pack/testhelpers"
	"github.com/buildpack/pack/testmocks"
)

func TestPackageBuilder(t *testing.T) {
	color.Disable(true)
	defer func() { color.Disable(false) }()
	spec.Run(t, "PackageBuilder", testPackageBuilder, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testPackageBuilder(t *testing.T, when spec.G, it spec.S) {
	var (
		fakePackageImage *fakes.Image
		mockController   *gomock.Controller
		mockImageFactory *testmocks.MockImageFactory
		subject          *buildpackage.PackageBuilder
	)

	it.Before(func() {
		mockController = gomock.NewController(t)
		mockImageFactory = testmocks.NewMockImageFactory(mockController)

		fakePackageImage = fakes.NewImage("some/package", "", "")
		mockImageFactory.EXPECT().NewImage("some/package", true).Return(fakePackageImage, nil).AnyTimes()

		subject = buildpackage.NewBuilder(mockImageFactory)
	})

	it.After(func() {
		mockController.Finish()
	})

	when("#Save", func() {
		when("validate default", func() {
			when("default not set", func() {
				it("returns error", func() {
					_, err := subject.Save(fakePackageImage.Name(), false)
					h.AssertError(t, err, "a default buildpack must be set")
				})
			})

			when("default is missing from blobs", func() {
				it("returns error", func() {
					subject.SetDefaultBuildpack(dist.BuildpackInfo{
						ID:      "buildpack.1.id",
						Version: "buildpack.1.version",
					})

					_, err := subject.Save(fakePackageImage.Name(), false)
					h.AssertError(t, err, "selected default 'buildpack.1.id@buildpack.1.version' is not present")
				})
			})
		})

		when("validate stacks", func() {
			it.Before(func() {
				buildpack, err := ifakes.NewBuildpackFromDescriptor(dist.BuildpackDescriptor{
					API: api.MustParse("0.2"),
					Info: dist.BuildpackInfo{
						ID:      "buildpack.1.id",
						Version: "buildpack.1.version",
					},
					Stacks: []dist.Stack{
						{ID: "stack.id.1"},
					},
					Order: nil,
				}, 0644)
				h.AssertNil(t, err)

				subject.SetDefaultBuildpack(dist.BuildpackInfo{
					ID:      buildpack.Descriptor().Info.ID,
					Version: buildpack.Descriptor().Info.Version,
				})

				subject.AddBuildpack(buildpack)
			})

			when("no stacks are set", func() {
				it("returns error", func() {
					_, err := subject.Save(fakePackageImage.Name(), false)
					h.AssertError(t, err, "must specify at least one supported stack")
				})
			})

			when("stack is added more than once", func() {
				it("should error", func() {
					subject.AddStack(dist.Stack{ID: "stack.id.1"})
					subject.AddStack(dist.Stack{ID: "stack.id.1"})

					_, err := subject.Save(fakePackageImage.Name(), false)
					h.AssertError(t, err, "stack 'stack.id.1' was specified more than once")
				})
			})

			when("stack is not listed in bp", func() {
				it("should error", func() {
					subject.AddStack(dist.Stack{ID: "stack.id.1"})
					subject.AddStack(dist.Stack{ID: "stack.id.not-supported-by-bps"})

					_, err := subject.Save(fakePackageImage.Name(), false)
					h.AssertError(t, err,
						"stack 'stack.id.not-supported-by-bps' is not supported by buildpack 'buildpack.1.id@buildpack.1.version'",
					)
				})
			})

			when("bp has more supported stacks than package supports", func() {
				it("should be successful", func() {
					buildpack2, err := ifakes.NewBuildpackFromDescriptor(dist.BuildpackDescriptor{
						API: api.MustParse("0.2"),
						Info: dist.BuildpackInfo{
							ID:      "buildpack.2.id",
							Version: "buildpack.2.version",
						},
						Stacks: []dist.Stack{
							{ID: "stack.id.1"},
							{ID: "stack.id.2"},
						},
						Order: nil,
					}, 0644)
					h.AssertNil(t, err)

					subject.AddBuildpack(buildpack2)
					subject.AddStack(dist.Stack{ID: "stack.id.1"})

					_, err = subject.Save(fakePackageImage.Name(), false)
					h.AssertNil(t, err)
				})
			})
		})
	})
}
