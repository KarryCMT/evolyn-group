import apps from './app';
import auths from './auth';
import dashboards from './dashboard';
import forms from './form';
import tenants from './tenant';
import workflows from './workflow';

export default [...auths, ...dashboards, ...tenants, ...apps, ...forms, ...workflows];
