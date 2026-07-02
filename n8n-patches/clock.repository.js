"use strict";
var __decorate = (this && this.__decorate) || function (decorators, target, key, desc) {
    var c = arguments.length, r = c < 3 ? target : desc === null ? desc = Object.getOwnPropertyDescriptor(target, key) : desc, d;
    if (typeof Reflect === "object" && typeof Reflect.decorate === "function") r = Reflect.decorate(decorators, target, key, desc);
    else for (var i = decorators.length - 1; i >= 0; i--) if (d = decorators[i]) r = (c < 3 ? d(r) : c > 3 ? d(target, key, r) : d(target, key)) || r;
    return c > 3 && r && Object.defineProperty(target, key, r), r;
};
var __metadata = (this && this.__metadata) || function (k, v) {
    if (typeof Reflect === "object" && typeof Reflect.metadata === "function") return Reflect.metadata(k, v);
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.ClockRepository = void 0;
const config_1 = require("@n8n/config");
const di_1 = require("@n8n/di");
const typeorm_1 = require("@n8n/typeorm");
const n8n_workflow_1 = require("n8n-workflow");
let ClockRepository = class ClockRepository {
    constructor(dataSource, databaseConfig) {
        this.dataSource = dataSource;
        this.databaseConfig = databaseConfig;
    }
    async getDbTime() {
        if (this.databaseConfig.type === 'postgresdb') {
            const [{ now }] = await this.dataSource.query('SELECT CURRENT_TIMESTAMP(3) AS now');
            return now instanceof Date ? now : new Date(now);
        }
        const [{ now }] = await this.dataSource.query("SELECT STRFTIME('%Y-%m-%dT%H:%M:%fZ', 'NOW') AS now");
        const date = new Date(now);
        if (Number.isNaN(date.getTime())) {
            throw new n8n_workflow_1.UnexpectedError(`Invalid DB server time: ${now}`);
        }
        return date;
    }
};
exports.ClockRepository = ClockRepository;
exports.ClockRepository = ClockRepository = __decorate([
    (0, di_1.Service)(),
    __metadata("design:paramtypes", [typeorm_1.DataSource,
        config_1.DatabaseConfig])
], ClockRepository);
//# sourceMappingURL=clock.repository.js.map