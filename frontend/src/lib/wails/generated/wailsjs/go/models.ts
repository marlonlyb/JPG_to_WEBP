export namespace app {
	
	export interface AppErrorDTO {
	    code: string;
	    message: string;
	    details?: string;
	}
	export interface BatchConvertRequestDTO {
	    inputs: string[];
	    overwrite: boolean;
	}
	export interface BatchSummaryDTO {
	    totalInputs: number;
	    completedInputs: number;
	    failedInputs: number;
	    totalOutputs: number;
	    writtenOutputs: number;
	    overwrittenOutputs: number;
	}
	export interface ConvertResultDTO {
	    outputPath: string;
	    outputBytes: number;
	    quality: number;
	    overwritten: boolean;
	}
	export interface ImageInfoDTO {
	    inputPath: string;
	    fileName: string;
	    width: number;
	    height: number;
	    inputBytes: number;
	}
	export interface BatchItemResultDTO {
	    input: ImageInfoDTO;
	    outputs: ConvertResultDTO[];
	    status: string;
	    error?: AppErrorDTO;
	}
	export interface BatchConvertResultDTO {
	    items: BatchItemResultDTO[];
	    summary: BatchSummaryDTO;
	}
	export interface OutputVariantDTO {
	    suffix: string;
	    quality: number;
	    outputPath: string;
	    exists: boolean;
	}
	export interface BatchInspectItemDTO {
	    input: ImageInfoDTO;
	    outputs: OutputVariantDTO[];
	}
	export interface BatchInspectionDTO {
	    items: BatchInspectItemDTO[];
	    totalInputs: number;
	    totalPlannedOutputs: number;
	}
	
	export interface BatchPreflightDTO {
	    conflicts: string[];
	    totalConflicts: number;
	    needsOverwrite: boolean;
	}
	
	export interface ConvertRequestDTO {
	    inputPath: string;
	    outputPath?: string;
	    quality: number;
	    overwrite: boolean;
	}
	
	

}

