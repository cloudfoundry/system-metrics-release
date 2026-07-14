require 'bosh/template/test'
require 'yaml'

def render_system_metrics_monit(properties = {})
  job_dir = File.join(File.dirname(__FILE__), '../../jobs/loggr-system-metrics-agent')
  spec = YAML.safe_load(File.read(File.join(job_dir, 'spec')))
  Bosh::Template::Test::Template.new(spec, File.join(job_dir, 'monit')).render(properties)
end

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

  describe 'monit' do
    context 'when enabled is true (default) and use_bpm is false (default)' do
      let(:rendered) { render_system_metrics_monit({}) }

      it 'uses the ctl script pidfile and start/stop programs' do
        expect(rendered).to include('with pidfile /var/vcap/sys/run/system-metrics-agent/system-metrics-agent.pid')
        expect(rendered).to include('start program "/var/vcap/jobs/loggr-system-metrics-agent/bin/ctl start"')
        expect(rendered).to include('stop program "/var/vcap/jobs/loggr-system-metrics-agent/bin/ctl stop"')
        expect(rendered).not_to include('/var/vcap/jobs/bpm/bin/bpm')
      end
    end

    context 'when enabled is true and use_bpm is true' do
      let(:rendered) { render_system_metrics_monit({'use_bpm' => true}) }

      it 'uses the bpm pidfile and start/stop programs' do
        expect(rendered).to include('with pidfile /var/vcap/sys/run/bpm/loggr-system-metrics-agent/loggr-system-metrics-agent.pid')
        expect(rendered).to include('start program "/var/vcap/jobs/bpm/bin/bpm start loggr-system-metrics-agent"')
        expect(rendered).to include('stop program "/var/vcap/jobs/bpm/bin/bpm stop loggr-system-metrics-agent"')
        expect(rendered).not_to include('/bin/ctl')
      end
    end

    context 'when enabled is false' do
      let(:rendered) { render_system_metrics_monit({'enabled' => false}) }

      it 'renders an empty monit file' do
        expect(rendered.strip).to be_empty
      end
    end
  end
end
