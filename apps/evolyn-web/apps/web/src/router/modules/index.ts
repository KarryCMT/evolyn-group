import auths from './auth';
import dashboards from './dashboard';
import tenants from './tenant';
import apps from './app';

export default [...auths, ...dashboards, ...tenants, ...apps];
