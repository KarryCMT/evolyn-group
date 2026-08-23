import auths from './auth';
import dashboards from './dashboard';
import tenants from './tenant';
import apps from './app';
import forms from './form';

export default [...auths, ...dashboards, ...tenants, ...apps, ...forms];
