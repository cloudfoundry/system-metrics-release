require 'bosh/template/test'
require 'yaml'

describe 'loggr-system-metrics-agent' do
  let(:release) { Bosh::Template::Test::ReleaseDir.new(File.join(File.dirname(__FILE__), '../..')) }
  let(:job) { release.job('loggr-system-metrics-agent') }

  describe 'config/bpm.yml' do
    let(:template) { job.template('config/bpm.yml') }
    let(:rendered) { template.render({}) }
    let(:bpm) { YAML.safe_load(rendered) }
    let(:processes) { bpm.fetch('processes') }
    let(:process) { processes.first }

    it 'defines exactly one process named loggr-system-metrics-agent' do
      expect(processes.length).to eq(1)
      expect(process['name']).to eq('loggr-system-metrics-agent')
    end

    it 'runs the system-metrics-agent binary' do
      expect(process['executable']).to eq(
        '/var/vcap/packages/system-metrics-agent/system-metrics-agent'
      )
    end

    it 'passes cert paths as environment variables' do
      env = process['env']
      expect(env['CA_CERT_PATH']).to include('system_metrics_agent_ca.crt')
      expect(env['CERT_PATH']).to include('system_metrics_agent.crt')
      expect(env['KEY_PATH']).to include('system_metrics_agent.key')
    end

    it 'passes the sample interval from properties' do
      expect(process['env']['SAMPLE_INTERVAL']).to eq('15s')
    end
  end
end
