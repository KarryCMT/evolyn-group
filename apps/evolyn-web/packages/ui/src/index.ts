import type { App } from 'vue';
import {
  EvolynButton,
  EvolynChart,
  EvolynGrid,
  EvolynMemberDepartmentRolePicker,
  EvolynTable,
} from './components';

export { version } from './version';

const components = [
  EvolynButton,
  EvolynChart,
  EvolynGrid,
  EvolynMemberDepartmentRolePicker,
  EvolynTable,
];

function install(app: App) {
  components.forEach((component) => {
    app.use(component);
  });
}

export { install };

export * from './components';

export default {
  install,
};
