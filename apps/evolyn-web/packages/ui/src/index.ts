import type { App } from 'vue';
import {
  EvolynGrid,
  EvolynMemberDepartmentRolePicker,
  EvolynTable,
  EvolynButton,
} from './components';

export { version } from './version';

const components = [EvolynButton, EvolynGrid, EvolynMemberDepartmentRolePicker, EvolynTable];

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
