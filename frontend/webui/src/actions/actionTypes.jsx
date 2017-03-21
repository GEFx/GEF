/**
 * Created by wqiu on 18/08/16.
 */
let actionTypes = {
    PAGE_CHANGE: 'PAGE_CHANGE',
    ERROR_OCCUR: 'ERROR_OCCUR',

    SERVICES_FETCH_START: 'SERVICES_FETCH_START',
    SERVICES_FETCH_SUCCESS: 'SERVICES_FETCH_SUCCESS',
    SERVICES_FETCH_ERROR: 'SERVICES_FETCH_ERROR',
    SERVICE_FETCH_START: 'SERVICE_FETCH_START',
    SERVICE_FETCH_SUCCESS: 'SERVICE_FETCH_SUCCESS',
    SERVICE_FETCH_ERROR: 'SERVICE_FETCH_ERROR',

    JOB_LIST_FETCH_START: 'JOB_LIST_FETCH_START',
    JOB_LIST_FETCH_SUCCESS: 'JOB_LIST_FETCH_SUCCESS',
    JOB_LIST_FETCH_ERROR: 'JOB_LIST_FETCH_ERROR',

    JOB_REMOVAL_START: 'JOB_REMOVAL_START',
    JOB_REMOVAL_SUCCESS: 'JOB_REMOVAL_SUCCESS',
    JOB_REMOVAL_ERROR: 'JOB_REMOVAL_ERROR',

    NEW_BUILD: 'NEW_BUILD',
    NEW_BUILD_ERROR:  'NEW_BUILD_ERROR',

    VOLUME_FETCH_START: 'VOLUME_FETCH_START',
    VOLUME_FETCH_SUCCESS: 'VOLUME_FETCH_SUCCESS',
    VOLUME_FETCH_ERROR: 'VOLUME_FETCH_ERROR',

    FILE_UPLOAD_START: 'FILE_UPLOAD_START',
    FILE_UPLOAD_SUCCESS: 'FILE_UPLOAD_SUCCESS',
    FILE_UPLOAD_ERROR: 'FILE_UPLOAD_ERROR',

    INSPECT_VOLUME_START: 'INSPECT_VOLUME_START',
    INSPECT_VOLUME_SUCCESS: 'INSPECT_VOLUME_SUCCESS',
    INSPECT_VOLUME_EMPTY: 'INSPECT_VOLUME_EMPTY',
    INSPECT_VOLUME_ERROR: 'INSPECT_VOLUME_ERROR',

    CONSOLE_OUTPUT_FETCH_START: 'CONSOLE_OUTPUT_FETCH_START',
    CONSOLE_OUTPUT_FETCH_SUCCESS: 'CONSOLE_OUTPUT_FETCH_SUCCESS',
    CONSOLE_OUTPUT_FETCH_EMPTY: 'CONSOLE_OUTPUT_FETCH_EMPTY',
    CONSOLE_OUTPUT_FETCH_ERROR: 'CONSOLE_OUTPUT_FETCH_ERROR',
};


export default actionTypes;