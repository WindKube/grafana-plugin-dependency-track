// Jest setup provided by Grafana scaffolding
import './.config/jest-setup';

// React's scheduler keeps a MessagePort open under Node 24, which stops Jest from exiting.
delete global.MessageChannel;
