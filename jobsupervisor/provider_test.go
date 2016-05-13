package jobsupervisor_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/jobsupervisor"
	"github.com/pivotal-golang/clock"
	fakemonit "github.com/cloudfoundry/bosh-agent/jobsupervisor/monit/fakes"
	fakembus "github.com/cloudfoundry/bosh-agent/mbus/fakes"
	fakeplatform "github.com/cloudfoundry/bosh-agent/platform/fakes"
	boshdir "github.com/cloudfoundry/bosh-agent/settings/directories"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

func init() {
	Describe("provider", func() {
		var (
			platform              *fakeplatform.FakePlatform
			client                *fakemonit.FakeMonitClient
			logger                boshlog.Logger
			dirProvider           boshdir.Provider
			jobFailuresServerPort int
			handler               *fakembus.FakeHandler
			provider              Provider
			timeService           clock.Clock
		)

		BeforeEach(func() {
			platform = fakeplatform.NewFakePlatform()
			client = fakemonit.NewFakeMonitClient()
			logger = boshlog.NewLogger(boshlog.LevelNone)
			dirProvider = boshdir.NewProvider("/fake-base-dir")
			jobFailuresServerPort = 2825
			handler = &fakembus.FakeHandler{}
			timeService = clock.NewClock()

			provider = NewProvider(
				platform,
				client,
				logger,
				dirProvider,
				handler,
			)
		})

		It("provides a monit job supervisor", func() {
			actualSupervisor, err := provider.Get("monit")
			Expect(err).ToNot(HaveOccurred())

			expectedSupervisor := NewMonitJobSupervisor(
				platform.Fs,
				platform.Runner,
				client,
				logger,
				dirProvider,
				jobFailuresServerPort,
				MonitReloadOptions{
					MaxTries:               3,
					MaxCheckTries:          6,
					DelayBetweenCheckTries: 5 * time.Second,
				},
				timeService,
			)
			Expect(actualSupervisor).To(Equal(expectedSupervisor))
		})

		It("provides a dummy job supervisor", func() {
			actualSupervisor, err := provider.Get("dummy")
			Expect(err).ToNot(HaveOccurred())

			expectedSupervisor := NewDummyJobSupervisor()
			Expect(actualSupervisor).To(Equal(expectedSupervisor))
		})

		It("provides a dummy nats job supervisor", func() {
			actualSupervisor, err := provider.Get("dummy-nats")
			Expect(err).NotTo(HaveOccurred())

			expectedSupervisor := NewDummyNatsJobSupervisor(handler)
			Expect(actualSupervisor).To(Equal(expectedSupervisor))
		})

		It("returns an error when the supervisor is not found", func() {
			_, err := provider.Get("does-not-exist")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does-not-exist could not be found"))
		})
	})
}
