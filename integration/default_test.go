package integration_test

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testDefault(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually
		port       = "8080"

		pack   occam.Pack
		docker occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()
	})

	context("when building a simple app", func() {
		var (
			image     occam.Image
			container occam.Container
			name      string
			source    string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		context("when building the default_app", func() {
			it("serves up the index.html", func() {
				var err error
				source, err = occam.Source(filepath.Join("testdata", "default_app"))
				Expect(err).NotTo(HaveOccurred())

				var logs fmt.Stringer
				image, logs, err = pack.WithNoColor().Build.
					WithPullPolicy("never").
					WithBuildpacks(
						buildpack,
						config.Nginx,
						config.StaticRequire,
					).
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred(), logs.String())

				container, err = docker.Container.Run.
					WithEnv(map[string]string{"PORT": port}).
					WithPublish(port).
					Execute(image.ID)
				Expect(err).ToNot(HaveOccurred())

				Eventually(container).Should(Serve(ContainSubstring("deplo.io static buildpack")).WithEndpoint("/index.html"))
			})
		})

		context("when building the default_app", func() {
			it("serves up the correct caching headers", func() {
				var err error
				source, err = occam.Source(filepath.Join("testdata", "default_app"))
				Expect(err).NotTo(HaveOccurred())

				var logs fmt.Stringer
				image, logs, err = pack.WithNoColor().Build.
					WithPullPolicy("never").
					WithBuildpacks(
						buildpack,
						config.Nginx,
						config.StaticRequire,
					).
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred(), logs.String())

				container, err = docker.Container.Run.
					WithEnv(map[string]string{"PORT": port}).
					WithPublish(port).
					Execute(image.ID)
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() error {
					resp, err := http.Get(fmt.Sprintf("http://%s:%s", container.Host(), container.HostPort(port)))
					if err != nil {
						return err
					}

					if resp.Header.Get("last-modified") == "" {
						return fmt.Errorf("expected last-modified header to be set")
					}

					if resp.Header.Get("last-modified") == "Tue, 01 Jan 1980 00:00:01 GMT" {
						return fmt.Errorf("expected last-modified header to not match the modified time")
					}

					if resp.Header.Get("etag") != "" {
						return fmt.Errorf("expected etag header to not be set")
					}

					return nil
				}).ShouldNot(HaveOccurred())
			})
		})

		context("when building the public_app", func() {
			it("serves up the index.html", func() {
				var err error
				source, err = occam.Source(filepath.Join("testdata", "public_app"))
				Expect(err).NotTo(HaveOccurred())

				var logs fmt.Stringer
				image, logs, err = pack.WithNoColor().Build.
					WithPullPolicy("never").
					WithBuildpacks(
						buildpack,
						config.Nginx,
						config.StaticRequire,
					).
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred(), logs.String())

				container, err = docker.Container.Run.
					WithEnv(map[string]string{"PORT": port}).
					WithPublish(port).
					Execute(image.ID)
				Expect(err).ToNot(HaveOccurred())

				Eventually(container).Should(Serve(ContainSubstring("satic site served from public dir")).WithEndpoint("/index.html"))
			})
		})

		context("when building the react_app", func() {
			it("serves up the built index.html", func() {
				var err error
				source, err = occam.Source(filepath.Join("testdata", "react_app"))
				Expect(err).NotTo(HaveOccurred())

				var logs fmt.Stringer
				image, logs, err = pack.WithNoColor().Build.
					WithPullPolicy("never").
					WithBuildpacks(
						buildpack,
						config.WebServers,
						config.StaticRequire,
					).
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred(), logs.String())

				container, err = docker.Container.Run.
					WithEnv(map[string]string{"PORT": port}).
					WithPublish(port).
					Execute(image.ID)
				Expect(err).ToNot(HaveOccurred())

				// we expect the string %PUBLIC% to not be in the index.html
				// anymore since npm build should take care of replacing that
				// during build.
				Eventually(container).Should(Serve(Not(ContainSubstring("%PUBLIC%"))).WithEndpoint("/index.html"))
			})
		})
	})
}
