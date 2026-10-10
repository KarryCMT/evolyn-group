import { createApp, defineComponent } from 'vue';
import DashboardDesignerToolbar from '~/components/dashboard/designer/DashboardDesignerToolbar.vue';
import '~/styles/index.scss';
import '@evolyn.do/ui/style.css';

const QaToolbar = defineComponent({
  components: { DashboardDesignerToolbar },
  template: `
    <main class="qa-page">
      <DashboardDesignerToolbar
        name="未命名仪表盘"
        :dirty="true"
        save-status="idle"
      />
      <div class="qa-center-line" aria-hidden="true"></div>
    </main>
  `,
});

createApp(QaToolbar).mount('#app');
