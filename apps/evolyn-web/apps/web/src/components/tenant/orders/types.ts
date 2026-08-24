/** 订单列表当前支持的筛选状态。 */
export type OrderStatusFilter = 'all' | 'pending-payment' | 'paid' | 'closed';

/** 发票列表当前支持的筛选状态。 */
export type InvoiceStatusFilter = 'all' | 'not-applied' | 'processing' | 'issued';
