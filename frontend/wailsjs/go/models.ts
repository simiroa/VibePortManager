export namespace config {
	
	export class Server {
	    id: string;
	    name: string;
	    command: string;
	    port: number;
	    autostart: boolean;
	    autorestart?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Server(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.command = source["command"];
	        this.port = source["port"];
	        this.autostart = source["autostart"];
	        this.autorestart = source["autorestart"];
	    }
	}
	export class Project {
	    id: string;
	    name: string;
	    path: string;
	    execution_target: string;
	    wsl_distro?: string;
	    package_manager: string;
	    servers: Server[];
	
	    static createFrom(source: any = {}) {
	        return new Project(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.execution_target = source["execution_target"];
	        this.wsl_distro = source["wsl_distro"];
	        this.package_manager = source["package_manager"];
	        this.servers = this.convertValues(source["servers"], Server);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace ports {
	
	export class PortEntry {
	    port: number;
	    pid: number;
	    processName: string;
	    backendId: string;
	
	    static createFrom(source: any = {}) {
	        return new PortEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.port = source["port"];
	        this.pid = source["pid"];
	        this.processName = source["processName"];
	        this.backendId = source["backendId"];
	    }
	}

}

export namespace project {
	
	export class DetectedServer {
	    name: string;
	    command: string;
	    port: number;
	
	    static createFrom(source: any = {}) {
	        return new DetectedServer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.command = source["command"];
	        this.port = source["port"];
	    }
	}
	export class ProjectAnalysis {
	    port?: number;
	    command: string;
	    scriptName: string;
	    packageMgr: string;
	    servers: DetectedServer[];
	
	    static createFrom(source: any = {}) {
	        return new ProjectAnalysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.port = source["port"];
	        this.command = source["command"];
	        this.scriptName = source["scriptName"];
	        this.packageMgr = source["packageMgr"];
	        this.servers = this.convertValues(source["servers"], DetectedServer);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

